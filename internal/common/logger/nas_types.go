package logger

import "github.com/reogac/nas"

// NasMsgTypeName returns a human-readable name for a NAS message type
func NasMsgTypeName(msgType uint8) string {
	switch msgType {
	// 5GMM Registration messages
	case nas.RegistrationRequestMsgType:
		return "RegistrationRequest"
	case nas.RegistrationAcceptMsgType:
		return "RegistrationAccept"
	case nas.RegistrationRejectMsgType:
		return "RegistrationReject"
	case nas.RegistrationCompleteMsgType:
		return "RegistrationComplete"

	// Authentication messages
	case nas.AuthenticationRequestMsgType:
		return "AuthenticationRequest"
	case nas.AuthenticationResponseMsgType:
		return "AuthenticationResponse"
	case nas.AuthenticationRejectMsgType:
		return "AuthenticationReject"
	case nas.AuthenticationFailureMsgType:
		return "AuthenticationFailure"

	// Identity messages
	case nas.IdentityRequestMsgType:
		return "IdentityRequest"
	case nas.IdentityResponseMsgType:
		return "IdentityResponse"

	// Security Mode messages
	case nas.SecurityModeCommandMsgType:
		return "SecurityModeCommand"
	case nas.SecurityModeCompleteMsgType:
		return "SecurityModeComplete"
	case nas.SecurityModeRejectMsgType:
		return "SecurityModeReject"

	// Service Request messages
	case nas.ServiceRequestMsgType:
		return "ServiceRequest"
	case nas.ServiceAcceptMsgType:
		return "ServiceAccept"
	case nas.ServiceRejectMsgType:
		return "ServiceReject"

	// Deregistration messages
	case nas.DeregistrationRequestFromUeMsgType:
		return "DeregistrationRequestFromUE"
	case nas.DeregistrationAcceptFromUeMsgType:
		return "DeregistrationAcceptFromUE"
	case nas.DeregistrationRequestToUeMsgType:
		return "DeregistrationRequestToUE"
	case nas.DeregistrationAcceptToUeMsgType:
		return "DeregistrationAcceptToUE"

	// Configuration Update messages
	case nas.ConfigurationUpdateCommandMsgType:
		return "ConfigurationUpdateCommand"
	case nas.ConfigurationUpdateCompleteMsgType:
		return "ConfigurationUpdateComplete"

	// NAS Transport messages
	case nas.UlNasTransportMsgType:
		return "UlNasTransport"
	case nas.DlNasTransportMsgType:
		return "DlNasTransport"

	// PDU Session messages
	case nas.PduSessionEstablishmentRequestMsgType:
		return "PduSessionEstablishmentRequest"
	case nas.PduSessionEstablishmentAcceptMsgType:
		return "PduSessionEstablishmentAccept"
	case nas.PduSessionEstablishmentRejectMsgType:
		return "PduSessionEstablishmentReject"
	case nas.PduSessionModificationRequestMsgType:
		return "PduSessionModificationRequest"
	case nas.PduSessionModificationCommandMsgType:
		return "PduSessionModificationCommand"
	case nas.PduSessionModificationRejectMsgType:
		return "PduSessionModificationReject"
	case nas.PduSessionModificationCompleteMsgType:
		return "PduSessionModificationComplete"
	case nas.PduSessionReleaseRequestMsgType:
		return "PduSessionReleaseRequest"
	case nas.PduSessionReleaseCommandMsgType:
		return "PduSessionReleaseCommand"
	case nas.PduSessionReleaseCompleteMsgType:
		return "PduSessionReleaseComplete"
	case nas.PduSessionReleaseRejectMsgType:
		return "PduSessionReleaseReject"

	// Status messages
	case nas.GmmStatusMsgType:
		return "5GMMStatus"
	case nas.GsmStatusMsgType:
		return "5GSMStatus"

	// Notification messages
	case nas.NotificationMsgType:
		return "Notification"
	case nas.NotificationResponseMsgType:
		return "NotificationResponse"

	default:
		return "Unknown"
	}
}

// NasRequestToResponses maps UE-sent NAS request to expected responses from 5G core
// Key: NAS message sent FROM UE TO 5G core
// Value: NAS messages sent FROM 5G core TO UE (valid responses)
var NasRequestToResponses = map[string][]string{
	"RegistrationRequest":            {"RegistrationReject", "IdentityRequest", "AuthenticationRequest"},
	"IdentityResponse":               {"AuthenticationRequest"},
	"AuthenticationFailure":          {"AuthenticationRequest", "AuthenticationReject"},
	"AuthenticationResponse":         {"SecurityModeCommand"},
	"SecurityModeReject":             {},
	"SecurityModeComplete":           {"RegistrationAccept"},
	"RegistrationComplete":           {"ConfigurationUpdateCommand"},
	"PduSessionEstablishmentRequest": {"PduSessionEstablishmentAccept", "PduSessionEstablishmentReject"},
	"DeregistrationRequestFromUE":    {"DeregistrationAcceptFromUE"},
	"ServiceRequest":                 {"ServiceAccept", "ServiceReject"},
}

// NasResponseToRequests maps received NAS response to expected pending request (reverse lookup)
// Key: NAS message received FROM 5G core
// Value: NAS messages that UE could have sent before this
var NasResponseToRequests = map[string][]string{
	"RegistrationReject":            {"RegistrationRequest"},
	"IdentityRequest":               {"RegistrationRequest"},
	"AuthenticationRequest":         {"RegistrationRequest", "IdentityResponse", "AuthenticationFailure"},
	"AuthenticationReject":          {"AuthenticationFailure"},
	"SecurityModeCommand":           {"AuthenticationResponse"},
	"RegistrationAccept":            {"SecurityModeComplete"},
	"ConfigurationUpdateCommand":    {"RegistrationComplete"},
	"DeregistrationAcceptFromUE":    {"DeregistrationRequestFromUE"},
	"ServiceAccept":                 {"ServiceRequest"},
	"ServiceReject":                 {"ServiceRequest"},
	"PduSessionEstablishmentAccept": {"PduSessionEstablishmentRequest"},
	"PduSessionEstablishmentReject": {"PduSessionEstablishmentRequest"},
}

// ValidNasPairs is kept for backward compatibility - maps response to request
var ValidNasPairs = NasResponseToRequests
