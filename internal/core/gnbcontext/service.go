package gnbcontext

import (
	"fmt"
	"stormsim/internal/common/pool"
	"stormsim/internal/transport/rlink"
	"stormsim/internal/transport/sctpngap"
	"stormsim/pkg/model"

	"github.com/lvdund/ngap/ies"
)

/*********************  AMF-SCTP connection *********************/
func (gnb *GnbContext) initConn(amf *GnbAmfContext) error {

	// check AMF IP and AMF port.
	remote := fmt.Sprintf("%s:%d", amf.amfIp, amf.amfPort)
	
	gnb.connMutex.Lock()
	localPort := gnb.controlPlaneInfo.gnbPort + gnb.connCount
	gnb.connCount++
	gnb.connMutex.Unlock()
	
	local := fmt.Sprintf("%s:%d", gnb.controlPlaneInfo.gnbIp, localPort)

	conn := sctpngap.NewSctpConn(gnb.GetId(), local, remote, gnb.ctx)
	if err := conn.Connect(); err != nil {
		gnb.Fatal("Create SCTP connection err:", err)
	}
	amf.tlnaAssoc.sctpConn = conn
	gnb.controlPlaneInfo.n2 = conn

	gnb.sctpListen(amf)

	return nil
}

func (gnb *GnbContext) sctpListen(amf *GnbAmfContext) {
	go func() {
		for rawMsg := range amf.tlnaAssoc.sctpConn.Read() {
			//logger.SctpConnStats[gnb.GetId()].DLmessages.Add(1)
			pool.GnbWorkerPool.Submit(func() { gnb.dispatch(amf, rawMsg) })
		}
	}()
}

/********************* Ue & Gnb connection *********************/

// handle connection between gnb & gnb
// if connection between ue & gnb is closed, ue send to
func (gnb *GnbContext) gnbListen() {
	for msg := range gnb.controlPlaneInfo.inboundChannel {
		switch message := msg.(type) {
		case *model.RLinkCreateConnectionRequest: // init new ue
			ue, err := gnb.getGnbUeByPrUeId(message.PrUeId)
			if ue != nil {
				gnb.Info("UE with PrUeId %v already exists", message.PrUeId)
				return
			}

			ue, err = gnb.newGnBUe(message.Conn, message.PrUeId, message.Msin, message.Tmsi)
			if err != nil || ue == nil {
				gnb.Error("Failed to create UE: %s. Closing connection", err)
				if message.Conn != nil {
					message.Conn.Close()
				}
				return
			}

			go gnb.listenToUE(message.PrUeId, message.Msin, message.Conn)
			gnb.Info("Received incoming connection from new UE with PrUeId: %v", message.PrUeId)

			message.Conn.SendDownlink(&model.RLinkCreateConnectionResponse{
				Mcc:  gnb.controlPlaneInfo.mcc,
				Mnc:  gnb.controlPlaneInfo.mnc,
				Conn: message.Conn,
			})

			ue.context.PduSession = [16]*model.GnbPDUSessionContext{}

		case *model.RlinkUeReadyPaging:
			if !message.FetchPagedUEs {
				return
			}

			if message.Conn == nil {
				gnb.Error("Unable to send PagedUEs to UE: connection is nil")
				return
			}

			message.Conn.SendDownlink(&model.RlinkSetupPaging{
				PagedUEs: gnb.getPagedUEs(),
			})

		case *model.RLinkHandoverForwardUeContext:
			ue, err := gnb.getGnbUeByPrUeId(message.PrUeId)
			if message.UeCoreContext != nil { // xn handover
				var err error
				ue, err = gnb.newGnBUe(message.Conn, message.PrUeId, message.Msin, nil)
				if err != nil {
					gnb.Error("Failed to create UE during Xn handover: %s", err)
					message.Conn.Close()
					return
				}
				go gnb.listenToUE(message.PrUeId, message.Msin, message.Conn)

				gnb.Info("Received incoming Xn handover for UE PrUeId:%d from gnb %s",
					message.PrUeId, message.SourceGnbId)

				ue.state = UE_READY
				ue.CopyFromPreviousContext(&message.AmfUeNgapId, message.UeCoreContext)
				gnb.sendPathSwitchRequest(ue)
			} else if ue != nil && message.Conn != nil { // n2 handover
				go gnb.listenToUE(message.PrUeId, message.Msin, message.Conn)

				gnb.Info("Received incoming N2 handover for UE PrUeId:%d from gnb %s",
					message.PrUeId, message.SourceGnbId)

				ue.rlinkConn = message.Conn
				ue.state = UE_READY
				ue.sendMsgToUe(&model.RlinkSetupPduSessonCommand{
					PrUeId:         message.PrUeId,
					GNBPduSessions: ue.context.PduSession,
					GnbIp:          gnb.dataPlaneInfo.gnbIp,
				})

				gnb.sendHandoverNotify(ue)
			} else {
				gnb.Error("UE was not created succesfully: %s", err.Error())
			}

		default:
			gnb.Error("Received unknown message type in ServeUE: %T", message)
		}
	}
}

// listen ue msg from `rlink.Connection`
func (gnb *GnbContext) listenToUE(ueID int64, msin string, conn *rlink.Connection) {
	uplinkCh := conn.GetUplinkChan()
	for {
		select {
		case <-gnb.close:
			return
		case msg, ok := <-uplinkCh:
			if !ok {
				gnb.Warn("UE %s disconnected", msin)
				gnb.removeConnection(rlink.ConnectionKey(ueID, gnb.controlPlaneInfo.gnbId))
				//logger.RLinkConnStats[gnb.GetId()].MessageReceivedDropped.Add(1)
				return
			}
			//logger.RLinkConnStats[gnb.GetId()].MessageReceived.Add(1)

			// Handle message from UE
			pool.GnbWorkerPool.Submit(func() { gnb.handleUeMsg(msg) })
		}
	}
}

// handle messages from UE if exist connection between UE & Gnb
func (gnb *GnbContext) handleUeMsg(msg rlink.Message) {
	switch message := msg.(type) {
	case *model.RlinkRlinkHandoverPrepareResponse:
		gnb.Info("Cleaning up UE (PrUeId: %v) context on source gNB", message.PrUeId)
		ue, _ := gnb.getGnbUeByPrUeId(message.PrUeId)
		ue.state = UE_DOWN
		if message.IsXnHandover {
			gnb.deleteGnBUe(ue)
		}

	case *model.NasMsg:
		ue, _ := gnb.getGnbUeByPrUeId(message.PrUeId)
		ue.sendNasPdu(message.Nas, gnb)

	case *model.RlinkUeIdleInform:
		ue, _ := gnb.getGnbUeByPrUeId(message.PrUeId)
		gnb.sendUeContextReleaseRequest(ue, &ies.Cause{
			Choice: ies.CausePresentRadionetwork, RadioNetwork: &ies.CauseRadioNetwork{
				Value: ies.CauseRadioNetworkUserinactivity,
			},
		})

	default:
		gnb.Error("Received unknown message type from UE: %T", message)
	}
}
