package uecontext

import (
	"stormsim/internal/common/fsm"
	"stormsim/pkg/model"

	"github.com/reogac/nas"
)

func (ue *UeContext) handleNas_n1sm(nasMsg *nas.NasMessage) {
	gsm := nasMsg.Gsm
	if gsm == nil {
		ue.Fatal("Err in DL NAS Transport, N1Sm is missing")
	}

	if ue.enableFuzz {
		Capture.CaptureMsgToUe(int(gsm.MsgType))
	}

	switch gsm.MsgType {
	case nas.PduSessionEstablishmentAcceptMsgType:
		ue.LogReceive("nas", "PduSessionEstablishmentAccept")
		//taskLog.Task = "Task SM: handle Receiving PDU Session Establishment Accept msg"
		ue.handlePduSessionEstablishmentAccept(gsm.PduSessionEstablishmentAccept)

	case nas.PduSessionReleaseCommandMsgType:
		ue.LogReceive("nas", "PduSessionReleaseCommand")
		//taskLog.Task = fmt.Sprintf("Task SM: Receiving PDU Session Release Command for session id = %d", gsm.PduSessionReleaseCommand.GetSessionId())
		ue.handlePduSessionReleaseCommand(gsm.PduSessionReleaseCommand)

	case nas.PduSessionEstablishmentRejectMsgType:
		ue.LogReceive("nas", "PduSessionEstablishmentReject")
		pduSessionId := gsm.PduSessionEstablishmentReject.GetSessionId()
		ue.Error(
			"PDU Session Establishment Reject for session id %d 5GSM Cause: %s",
			pduSessionId,
			cause5GSMToString(uint8(gsm.PduSessionEstablishmentReject.GsmCause)),
		)
		//taskLog.Task = fmt.Sprintf(
		// "Task SM: Receiving PDU Session Establishment Reject for session id %d 5GSM Cause: %s",
		// pduSessionId,
		// cause5GSMToString(uint8(gsm.PduSessionEstablishmentReject.GsmCause)),
		// )
		ue.handlePduSessionEstablishmentReject(gsm.PduSessionEstablishmentReject)

	case nas.GsmStatusMsgType:
		ue.LogReceive("nas", "5GSMStatus")
		//taskLog.Task = "Task SM: handle Receive Status 5GSM msg"
		ue.handleCause5GSM(&gsm.GsmStatus.GsmCause)

	default:
		ue.Error("Receiving Unknown Dl NAS Transport message!! %d", gsm.MsgType)
		//taskLog.Task = fmt.Sprintf("Task SM: Receiving Unknown Dl NAS Transport message!! %d", gsm.MsgType)
	}
}

func (ue *UeContext) handlePduSessionEstablishmentAccept(msg *nas.PduSessionEstablishmentAccept) {

	if msg.GetPti() != 1 {
		ue.Fatal("Error in PDU Session Establishment Accept, PTI not the expected value")
	}
	if msg.SelectedPduSessionType != 1 {
		ue.Fatal("Error in PDU Session Establishment Accept, PDU Session Type not the expected value")
	}

	// update PDU Session information.
	pduSessionId := msg.GetSessionId()
	pduSession, err := ue.getPduSession(pduSessionId)
	if err != nil {
		ue.Error("Receiving PDU Session Establishment Accept about an unknown PDU Session, id: %d", pduSessionId)
		return
	}

	if msg.PduAddress != nil {
		UeIp := msg.PduAddress
		pduSession.setIp(UeIp.Content())
		pduSession.Info("PDU address received: %s", pduSession.ueIP)
	}

	// get QoS Rules
	QosRule := msg.AuthorizedQosRules
	pduSession.Info("PDU session QoS RULES: %v", QosRule.Bytes)

	// get DNN
	if msg.Dnn != nil {
		pduSession.Info("PDU session DNN: %s", msg.Dnn.String())
	}

	// get SNSSAI
	if msg.SNssai != nil {
		sst := msg.SNssai.Sst
		sd := msg.SNssai.GetSd()
		pduSession.Info("PDU session NSSAI -- sst:%d sd:%s", sst, sd)
	}

	pduSession.SendEventSm(fsm.NewEventData(model.EstablishmentAccept, msg))
}

func (ue *UeContext) handlePduSessionEstablishmentReject(msg *nas.PduSessionEstablishmentReject) {

	// Per 5GSM state machine in TS 24.501 - 6.1.3.2.1., we re-try the setup until it's successful
	pduSession, err := ue.getPduSession(msg.GetSessionId())
	if err != nil {
		pduSession.Error("Cannot retry PDU Session Request for PDU Session after Reject as %v", err)
		return
	}

	pduSession.SendEventSm(fsm.NewEmptyEventData(model.EstablishmentReject))
}

func (ue *UeContext) handlePduSessionReleaseCommand(msg *nas.PduSessionReleaseCommand) {
	pduSession, err := ue.getPduSession(msg.GetSessionId())
	if pduSession == nil || err != nil {
		pduSession.Error("Unable to delete PDU Session from UE as the PDU Session was not found. Ignoring.")
		return
	}

	pduSession.SendEventSm(fsm.NewEmptyEventData(model.ReleaseCommand))
}

func (ue *UeContext) handleCause5GSM(cause *uint8) {
	if cause != nil {
		ue.Error("UE received a 5GSM Failure, cause: %s", cause5GSMToString(uint8(*cause)))
	}
}
