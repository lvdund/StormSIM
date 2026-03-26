package logger

import (
	"reflect"

	"github.com/lvdund/ngap/ies"
)

// NgapMsgTypeName returns a human-readable name for an NGAP message type
// Uses type assertion on struct pointers from github.com/lvdund/ngap/ies
func NgapMsgTypeName(msg any) string {
	if msg == nil {
		return "Unknown"
	}

	// Use type switch on NGAP struct types
	switch msg.(type) {
	// Initiating Messages (AMF → gNB)
	case *ies.DownlinkNASTransport:
		return "DownlinkNASTransport"
	case *ies.InitialContextSetupRequest:
		return "InitialContextSetupRequest"
	case *ies.PDUSessionResourceSetupRequest:
		return "PDUSessionResourceSetupRequest"
	case *ies.PDUSessionResourceReleaseCommand:
		return "PDUSessionResourceReleaseCommand"
	case *ies.UEContextReleaseCommand:
		return "UEContextReleaseCommand"
	case *ies.AMFConfigurationUpdate:
		return "AMFConfigurationUpdate"
	case *ies.AMFStatusIndication:
		return "AMFStatusIndication"
	case *ies.HandoverRequest:
		return "HandoverRequest"
	case *ies.Paging:
		return "Paging"
	case *ies.ErrorIndication:
		return "ErrorIndication"

	// Initiating Messages (gNB → AMF)
	case *ies.NGSetupRequest:
		return "NGSetupRequest"
	case *ies.InitialUEMessage:
		return "InitialUEMessage"
	case *ies.UplinkNASTransport:
		return "UplinkNASTransport"
	case *ies.PathSwitchRequest:
		return "PathSwitchRequest"
	case *ies.HandoverNotify:
		return "HandoverNotify"
	case *ies.HandoverRequired:
		return "HandoverRequired"
	case *ies.UEContextReleaseRequest:
		return "UEContextReleaseRequest"

	// Successful Outcomes
	case *ies.NGSetupResponse:
		return "NGSetupResponse"
	case *ies.InitialContextSetupResponse:
		return "InitialContextSetupResponse"
	case *ies.PDUSessionResourceSetupResponse:
		return "PDUSessionResourceSetupResponse"
	case *ies.PDUSessionResourceReleaseResponse:
		return "PDUSessionResourceReleaseResponse"
	case *ies.UEContextReleaseComplete:
		return "UEContextReleaseComplete"
	case *ies.PathSwitchRequestAcknowledge:
		return "PathSwitchRequestAcknowledge"
	case *ies.HandoverCommand:
		return "HandoverCommand"
	case *ies.HandoverRequestAcknowledge:
		return "HandoverRequestAcknowledge"
	case *ies.AMFConfigurationUpdateAcknowledge:
		return "AMFConfigurationUpdateAcknowledge"

	// Unsuccessful Outcomes
	case *ies.NGSetupFailure:
		return "NGSetupFailure"
	case *ies.InitialContextSetupFailure:
		return "InitialContextSetupFailure"
	case *ies.HandoverFailure:
		return "HandoverFailure"
	case *ies.PathSwitchRequestFailure:
		return "PathSwitchRequestFailure"
	case *ies.AMFConfigurationUpdateFailure:
		return "AMFConfigurationUpdateFailure"

	default:
		// Try to get type name via reflection as fallback
		t := reflect.TypeOf(msg)
		if t != nil {
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			return t.Name()
		}
		return "Unknown"
	}
}

// NgapProcedureCodeName returns a human-readable name for NGAP procedure code
func NgapProcedureCodeName(procedureCode int64) string {
	switch procedureCode {
	case ies.ProcedureCode_NGSetup:
		return "NGSetup"
	case ies.ProcedureCode_InitialUEMessage:
		return "InitialUEMessage"
	case ies.ProcedureCode_UplinkNASTransport:
		return "UplinkNASTransport"
	case ies.ProcedureCode_DownlinkNASTransport:
		return "DownlinkNASTransport"
	case ies.ProcedureCode_InitialContextSetup:
		return "InitialContextSetup"
	case ies.ProcedureCode_PDUSessionResourceSetup:
		return "PDUSessionResourceSetup"
	case ies.ProcedureCode_PDUSessionResourceRelease:
		return "PDUSessionResourceRelease"
	case ies.ProcedureCode_UEContextRelease:
		return "UEContextRelease"
	case ies.ProcedureCode_Paging:
		return "Paging"
	case ies.ProcedureCode_PathSwitchRequest:
		return "PathSwitchRequest"
	case ies.ProcedureCode_HandoverPreparation:
		return "HandoverPreparation"
	case ies.ProcedureCode_HandoverResourceAllocation:
		return "HandoverResourceAllocation"
	case ies.ProcedureCode_HandoverNotification:
		return "HandoverNotification"
	case ies.ProcedureCode_AMFConfigurationUpdate:
		return "AMFConfigurationUpdate"
	case ies.ProcedureCode_AMFStatusIndication:
		return "AMFStatusIndication"
	case ies.ProcedureCode_ErrorIndication:
		return "ErrorIndication"
	default:
		return "Unknown"
	}
}
