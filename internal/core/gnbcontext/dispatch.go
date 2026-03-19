package gnbcontext

import (
	"stormsim/internal/common/logger"

	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/ies"
)

// LogNgapReceive records receive timestamp and logs the NGAP message
func (gnb *GnbContext) LogNgapReceive(msgType string) {
	gnb.delayTracker.RecordReceive("ngap", msgType)
	gnb.Info("Receive %s", msgType)
}

// LogNgapSend records send timestamp and logs the NGAP message
func (gnb *GnbContext) LogNgapSend(msgType string) {
	gnb.delayTracker.RecordSend("ngap", msgType)
	gnb.Info("Send %s", msgType)
}

// GetDelayLogs returns NGAP delay log entries
func (gnb *GnbContext) GetDelayLogs(last int) []logger.DelayEntry {
	return gnb.delayTracker.GetLogs(last)
}

// GetDelayStats returns aggregated delay statistics
func (gnb *GnbContext) GetDelayStats() logger.DelayStats {
	return gnb.delayTracker.GetStats()
}

func (gnb *GnbContext) dispatch(amf *GnbAmfContext, ngapPdu []byte) {
	if len(ngapPdu) == 0 {
		gnb.Error("NGAP message is empty")
		return
	}

	ngapMsg, err, _ := ngap.NgapDecode(ngapPdu)
	if err != nil {
		gnb.Error("Error decoding NGAP message in %s GNB: %v", gnb.controlPlaneInfo.gnbId, err)
	}

	// var taskLog *logger.TaskStat = &logger.TaskStat{}
	//logger.TaskStats[gnb.GetId()] = append(logger.TaskStats[gnb.GetId()], taskLog)
	// taskLog.Start = time.Now()

	// handle NGAP message.
	switch ngapMsg.Present {

	case ies.NgapPduInitiatingMessage:

		switch ngapMsg.Message.ProcedureCode.Value {

		case ies.ProcedureCode_DownlinkNASTransport:
			gnb.LogNgapReceive("DownlinkNASTransport")
			//taskLog.Task = "Task: handle Receive Downlink NAS Transport msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.DownlinkNASTransport)
			gnb.handlerDownlinkNasTransport(innerMsg)

		case ies.ProcedureCode_InitialContextSetup:
			gnb.LogNgapReceive("InitialContextSetupRequest")
			//taskLog.Task = "Task: handle Receive Initial Context Setup Request msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.InitialContextSetupRequest)
			gnb.handlerInitialContextSetupRequest(innerMsg)

		case ies.ProcedureCode_PDUSessionResourceSetup:
			gnb.LogNgapReceive("PDUSessionResourceSetupRequest")
			//taskLog.Task = "Task: handle Receive PDU Session Resource Setup Request msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.PDUSessionResourceSetupRequest)
			gnb.handlerPduSessionResourceSetupRequest(innerMsg)

		case ies.ProcedureCode_PDUSessionResourceRelease:
			gnb.LogNgapReceive("PDUSessionResourceReleaseCommand")
			//taskLog.Task = "Task: handle Receive PDU Session Release Command msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.PDUSessionResourceReleaseCommand)
			gnb.handlerPduSessionReleaseCommand(innerMsg)

		case ies.ProcedureCode_UEContextRelease:
			gnb.LogNgapReceive("UEContextReleaseCommand")
			//taskLog.Task = "Task: handle Receive UE Context Release Command msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.UEContextReleaseCommand)
			gnb.handlerUeContextReleaseCommand(innerMsg)

		case ies.ProcedureCode_AMFConfigurationUpdate:
			gnb.LogNgapReceive("AMFConfigurationUpdate")
			//taskLog.Task = "Task: handle Receive AMF Configuration Update msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.AMFConfigurationUpdate)
			gnb.handlerAmfConfigurationUpdate(amf, innerMsg)
		case ies.ProcedureCode_AMFStatusIndication:
			gnb.LogNgapReceive("AMFStatusIndication")
			//taskLog.Task = "Task: handle Receive AMF Status Indication msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.AMFStatusIndication)
			gnb.handlerAmfStatusIndication(amf, innerMsg)
		case ies.ProcedureCode_HandoverResourceAllocation:
			gnb.LogNgapReceive("HandoverRequest")
			//taskLog.Task = "Task: handle Receive Handover Request msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.HandoverRequest)
			gnb.handlerHandoverRequest(amf, innerMsg)

		case ies.ProcedureCode_Paging:
			gnb.LogNgapReceive("Paging")
			//taskLog.Task = "Task: handle Receive Paging msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.Paging)
			gnb.handlerPaging(innerMsg)

		case ies.ProcedureCode_ErrorIndication:
			gnb.LogNgapReceive("ErrorIndication")
			//taskLog.Task = "Task: handle Receive Error Indication msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.ErrorIndication)
			gnb.handlerErrorIndication(innerMsg)

		default:
			gnb.Warn("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
			//taskLog.Task = fmt.Sprintf("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}

	case ies.NgapPduSuccessfulOutcome:

		switch ngapMsg.Message.ProcedureCode.Value {

		case ies.ProcedureCode_NGSetup:
			gnb.LogNgapReceive("NGSetupResponse")
			//taskLog.Task = "Task: handle Receive NG Setup Response msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.NGSetupResponse)
			gnb.handlerNgSetupResponse(amf, innerMsg)

		case ies.ProcedureCode_PathSwitchRequest:
			gnb.LogNgapReceive("PathSwitchRequestAcknowledge")
			//taskLog.Task = "Task: handle Receive PathSwitchRequestAcknowledge msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.PathSwitchRequestAcknowledge)
			gnb.handlerPathSwitchRequestAcknowledge(innerMsg)

		case ies.ProcedureCode_HandoverPreparation:
			gnb.LogNgapReceive("HandoverCommand")
			//taskLog.Task = "Task: handle Receive Handover Command msg"
			innerMsg := ngapMsg.Message.Msg.(*ies.HandoverCommand)
			gnb.handlerHandoverCommand(amf, innerMsg)

		default:
			gnb.Warn("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
			//taskLog.Task = fmt.Sprintf("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}

	case ies.NgapPduUnsuccessfulOutcome:

		switch ngapMsg.Message.ProcedureCode.Value {

		case ies.ProcedureCode_NGSetup:
			gnb.LogNgapReceive("NGSetupFailure")
			//taskLog.Task = "Task: handle Receive Ng Setup Failure msg"
			amf.state = AMF_INACTIVE
			gnb.Info("AMF is inactive")

		default:
			gnb.Warn("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
			//taskLog.Task = fmt.Sprintf("Received unknown NGAP message 0x%x", ngapMsg.Message.ProcedureCode.Value)
		}
	}

	// endTime := time.Now()
	// duration := endTime.Sub(taskLog.Start)
	// taskLog.End = &endTime
	// taskLog.Durarion = &duration
}
