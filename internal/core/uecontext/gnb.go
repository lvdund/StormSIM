package uecontext

import (
	"stormsim/internal/common/fsm"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"

	"github.com/reogac/nas"
	"github.com/reogac/sbi/models"
)

func (ue *UeContext) handleGnbMsg(msg rlink.Message) {
	if msg.GetType() == model.NasMsgType {
		message := msg.(*model.NasMsg)
		ue.sendEventMm(fsm.NewEventData(model.GmmMessageEvent, &message.Nas))
	} else if msg.GetType() == model.RlinkSetupPagingType {
		message := msg.(*model.RlinkSetupPaging)
		for _, pagedUE := range message.PagedUEs {
			if ue.guti != nil && pagedUE.FiveGSTMSI != nil &&
				[4]uint8(pagedUE.FiveGSTMSI.FiveGTMSI) == ue.getTmsiBytes() {
				ue.sendEventMm(fsm.NewEventData(model.ServiceRequestInit, ue))
				return
			}
		}
	} else if msg.GetType() == model.RlinkSetupPduSessonCommandType {
		if ue.tunnelMode == model.TunnelDisabled {
			// ue.Warn("[GTP]Interface has not been created: tunnel has been disabled")
		} else { // setup pdu session
			ue.setupGtpInterface(msg.(*model.RlinkSetupPduSessonCommand))
		}
	} else if msg.GetType() == model.RLinkHandoverPrepareRequestType {
		message := msg.(*model.RLinkHandoverPrepareRequest)
		if message.Conn != nil && message.TargetGnbId != "" {
			ue.Info("gNodeB-%s is asking to use another gNodeB", ue.gnbId)
			ue.sendGnb(&model.RlinkRlinkHandoverPrepareResponse{
				PrUeId:       int64(ue.id),
				IsXnHandover: message.IsXnHandover,
				IsN2Handover: message.IsN2Handover,
			})
			ue.rlinkConn.Close()

			ue.gnbId = message.TargetGnbId
			ue.rlinkConn = message.Conn
		}
	} else {
		ue.Error("Received unknown message from gNodeB: %v", msg)
	}
}

func (ue *UeContext) sendGnb(message rlink.Message) {
	ue.mutex.Lock()
	ue.rlinkConn.SendUplink(message)
	ue.mutex.Unlock()
}

func (ue *UeContext) sendN1Sm(
	n1Sm []byte,
	sessionId uint8,
	requestType *uint8,
	params *map[string]any,
) {
	msg := &nas.UlNasTransport{
		PayloadContainer:     n1Sm,
		PayloadContainerType: nas.PayloadContainerTypeN1SMInfo,
		PduSessionId:         &sessionId,
		SNssai:               new(nas.SNssai),
	}
	if requestType != nil {
		msg.RequestType = requestType
	}

	if val, ok := (*params)["dnn"]; ok && val != "" {
		msg.Dnn = nas.NewDnn(val.(string))
	} else if len(ue.dnn) > 0 {
		msg.Dnn = nas.NewDnn(ue.dnn)
	}

	if val, ok := (*params)["snssai"]; ok && val != nil {
		if s, ok := val.(models.Snssai); ok {
			msg.SNssai.Set(uint8(s.Sst), s.Sd)
		}
	} else {
		msg.SNssai.Set(uint8(ue.snssai.Sst), ue.snssai.Sd)
	}

	nasCtx := ue.getNasContext() //must be non nil
	msg.SetSecurityHeader(nas.NasSecBoth)
	if nasPdu, err := nas.EncodeMm(nasCtx, msg); err != nil {
		ue.Fatal("Error send N1 Sm: ul nas transport: %v", err)
	} else {
		ue.Info("send n1sm msg to amf")
		ue.sendNas(nasPdu) // sending to GNB
	}
}

func (ue *UeContext) sendNas(nasPdu []byte) {
	if ue.enableFuzz {
		Capture.CaptureMsgFromUe(ue.state_mm.CurrentState(), nasPdu)
	}
	ue.sendGnb(&model.NasMsg{PrUeId: int64(ue.id), Nas: nasPdu})
}
