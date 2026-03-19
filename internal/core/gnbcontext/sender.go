package gnbcontext

import (
	"fmt"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"time"
)

/**************** send to gnb listener ***************/
func SendToGnb(source string, msg rlink.Message, targetGnbId string, isInnerGnb bool) error {
	val, ok := Gnbs.Load(targetGnbId)
	if !ok {
		//logger.RLinkConnStats[source].MessageSendDropped.Add(1)
		return fmt.Errorf("Cannot find gnb %s", targetGnbId)
	}

	gnb := val.(*GnbContext)
	if gnb.controlPlaneInfo.inboundChannel == nil {
		//logger.RLinkConnStats[source].MessageSendDropped.Add(1)
		//logger.RLinkConnStats[targetGnbId].MessageReceivedDropped.Add(1)
		return fmt.Errorf("Cannot send %s msg to gnb %s",
			msg.GetType(), targetGnbId)
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		//logger.RLinkConnStats[source].MessageSendDropped.Add(1)
		//logger.RLinkConnStats[targetGnbId].MessageReceivedDropped.Add(1)
		return fmt.Errorf("Cannot send %s msg to gnb %s: timeout (1 second)",
			msg.GetType(), targetGnbId)
	case gnb.controlPlaneInfo.inboundChannel <- msg:
		//logger.RLinkConnStats[source].MessageSend.Add(1)
		//logger.RLinkConnStats[targetGnbId].MessageReceived.Add(1)
		return nil
	}
}

/******************** Send mg to UE *****************/

func (ue *GnbUeContext) sendMsgToUe(msg rlink.Message) {
	ue.rlinkConn.SendDownlink(msg)
}
func (ue *GnbUeContext) sendNasToUe(nasPdu []byte) {
	ue.sendMsgToUe(&model.NasMsg{PrUeId: ue.prUeId, Nas: nasPdu})
}

/******************** send ngap msg ********************/
func (ue *GnbUeContext) sendNasPdu(nasPdu []byte, gnb *GnbContext) {
	var ngap []byte
	var err error
	newState := ue.state
	switch ue.state {
	case UE_INITIALIZED:
		ngap, err = gnb.buildInitialUeMessage(nasPdu, ue)
		newState = UE_ONGOING
		if err != nil {
			gnb.Error("Error create Initial UE Message: %v", err)
			return
		}
		ue.Info("Sending Initial UE Message to AMF")

	case UE_ONGOING, UE_READY:
		ngap, err = gnb.buildUplinkNasTransport(nasPdu, ue)
		ue.Info("Sending Uplink Nas Transport to AMF")
		if err != nil {
			gnb.Error("Error create Uplink Nas Transport: %v", err)
			return
		}
	}
	ue.state = newState
	err = ue.sendNgap(ngap)
	if err != nil {
		gnb.Error("Error sending Nas message in NGAP: %v", err)
	}
}
func (ue *GnbUeContext) sendNgap(msg []byte) error {
	//TODO: included information for SCTP association.
	ue.sctpConnection.Send(msg)
	return nil
}
func (amf *GnbAmfContext) sendNgap(pdu []byte) error {
	//TODO: included information for SCTP association.
	amf.tlnaAssoc.sctpConn.Send(pdu)
	return nil
}
