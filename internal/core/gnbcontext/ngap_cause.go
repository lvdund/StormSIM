package gnbcontext

import (
	"github.com/lvdund/ngap/ies"
)

func causeToString(cause *ies.Cause) string {
	if cause != nil {
		switch cause.Choice {
		case uint64(ies.CausePresentRadionetwork):
			return "radioNetwork: " + causeRadioNetworkToString(cause.RadioNetwork)
		case uint64(ies.CausePresentTransport):
			return "transport: " + causeTransportToString(cause.Transport)
		case uint64(ies.CausePresentNas):
			return "nas: " + causeNasToString(cause.Nas)
		case uint64(ies.CausePresentProtocol):
			return "protocol: " + causeProtocolToString(cause.Protocol)
		case uint64(ies.CausePresentMisc):
			return "misc: " + causeMiscToString(cause.Misc)
		}
	}
	return "Cause not found"
}

func causeRadioNetworkToString(network *ies.CauseRadioNetwork) string {
	switch network.Value {
	case ies.CauseRadioNetworkUnspecified:
		return "Unspecified cause for radio network"
	case ies.CauseRadioNetworkTxnrelocoverallexpiry:
		return "Transfer the overall timeout of radio resources during handover"
	case ies.CauseRadioNetworkSuccessfulhandover:
		return "Successful handover"
	case ies.CauseRadioNetworkReleaseduetongrangeneratedreason:
		return "Release due to NG-RAN generated reason"
	case ies.CauseRadioNetworkReleasedueto5Gcgeneratedreason:
		return "Release due to 5GC generated reason"
	case ies.CauseRadioNetworkHandovercancelled:
		return "Handover cancelled"
	case ies.CauseRadioNetworkPartialhandover:
		return "Partial handover"
	case ies.CauseRadioNetworkHofailureintarget5Gcngrannodeortargetsystem:
		return "Handover failure in target 5GC NG-RAN node or target system"
	case ies.CauseRadioNetworkHotargetnotallowed:
		return "Handover target not allowed"
	case ies.CauseRadioNetworkTngrelocoverallexpiry:
		return "Transfer the overall timeout of radio resources during target NG-RAN relocation"
	case ies.CauseRadioNetworkTngrelocprepexpiry:
		return "Transfer the preparation timeout of radio resources during target NG-RAN relocation"
	case ies.CauseRadioNetworkCellnotavailable:
		return "Cell not available"
	case ies.CauseRadioNetworkUnknowntargetid:
		return "Unknown target ID"
	case ies.CauseRadioNetworkNoradioresourcesavailableintargetcell:
		return "No radio resources available in the target cell"
	case ies.CauseRadioNetworkUnknownlocaluengapid:
		return "Unknown local UE NGAP ID"
	case ies.CauseRadioNetworkInconsistentremoteuengapid:
		return "Inconsistent remote UE NGAP ID"
	case ies.CauseRadioNetworkHandoverdesirableforradioreason:
		return "Handover desirable for radio reason"
	case ies.CauseRadioNetworkTimecriticalhandover:
		return "Time-critical handover"
	case ies.CauseRadioNetworkResourceoptimisationhandover:
		return "Resource optimization handover"
	case ies.CauseRadioNetworkReduceloadinservingcell:
		return "Reduce load in serving cell"
	case ies.CauseRadioNetworkUserinactivity:
		return "User inactivity"
	case ies.CauseRadioNetworkRadioconnectionwithuelost:
		return "Radio connection with UE lost"
	case ies.CauseRadioNetworkRadioresourcesnotavailable:
		return "Radio resources not available"
	case ies.CauseRadioNetworkInvalidqoscombination:
		return "Invalid QoS combination"
	case ies.CauseRadioNetworkFailureinradiointerfaceprocedure:
		return "Failure in radio interface procedure"
	case ies.CauseRadioNetworkInteractionwithotherprocedure:
		return "Interaction with other procedure"
	case ies.CauseRadioNetworkUnknownpdusessionid:
		return "Unknown PDU session ID"
	case ies.CauseRadioNetworkUnkownqosflowid:
		return "Unknown QoS flow ID"
	case ies.CauseRadioNetworkMultiplepdusessionidinstances:
		return "Multiple PDU session ID instances"
	case ies.CauseRadioNetworkMultipleqosflowidinstances:
		return "Multiple QoS flow ID instances"
	case ies.CauseRadioNetworkEncryptionandorintegrityprotectionalgorithmsnotsupported:
		return "Encryption and/or integrity protection algorithms not supported"
	case ies.CauseRadioNetworkNgintrasystemhandovertriggered:
		return "NG intra-system handover triggered"
	case ies.CauseRadioNetworkNgintersystemhandovertriggered:
		return "NG inter-system handover triggered"
	case ies.CauseRadioNetworkXnhandovertriggered:
		return "Xn handover triggered"
	case ies.CauseRadioNetworkNotsupported5Qivalue:
		return "Not supported 5QI value"
	case ies.CauseRadioNetworkUecontexttransfer:
		return "UE context transfer"
	case ies.CauseRadioNetworkImsvoiceepsfallbackorratfallbacktriggered:
		return "IMS voice EPS fallback or RAT fallback triggered"
	case ies.CauseRadioNetworkUpintegrityprotectionnotpossible:
		return "UP integrity protection not possible"
	case ies.CauseRadioNetworkUpconfidentialityprotectionnotpossible:
		return "UP confidentiality protection not possible"
	case ies.CauseRadioNetworkSlicenotsupported:
		return "Slice not supported"
	case ies.CauseRadioNetworkUeinrrcinactivestatenotreachable:
		return "UE in RRC inactive state not reachable"
	case ies.CauseRadioNetworkRedirection:
		return "Redirection"
	case ies.CauseRadioNetworkResourcesnotavailablefortheslice:
		return "Resources not available for the slice"
	case ies.CauseRadioNetworkUemaxintegrityprotecteddataratereason:
		return "UE maximum integrity protected data rate reason"
	case ies.CauseRadioNetworkReleaseduetocndetectedmobility:
		return "Release due to CN detected mobility"
	default:
		return "Unknown cause for radio network"
	}
}

func causeTransportToString(transport *ies.CauseTransport) string {
	switch transport.Value {
	case ies.CauseTransportTransportresourceunavailable:
		return "Transport resource unavailable"
	case ies.CauseTransportUnspecified:
		return "Unspecified cause for transport"
	default:
		return "Unknown cause for transport"
	}
}

func causeNasToString(nas *ies.CauseNas) string {
	switch nas.Value {
	case ies.CauseNasNormalrelease:
		return "Normal release"
	case ies.CauseNasAuthenticationfailure:
		return "Authentication failure"
	case ies.CauseNasDeregister:
		return "Deregister"
	case ies.CauseNasUnspecified:
		return "Unspecified cause for NAS"
	default:
		return "Unknown cause for NAS"
	}
}

func causeProtocolToString(protocol *ies.CauseProtocol) string {
	switch protocol.Value {
	case ies.CauseProtocolTransfersyntaxerror:
		return "Transfer syntax error"
	case ies.CauseProtocolAbstractsyntaxerrorreject:
		return "Abstract syntax error - Reject"
	case ies.CauseProtocolAbstractsyntaxerrorignoreandnotify:
		return "Abstract syntax error - Ignore and notify"
	case ies.CauseProtocolMessagenotcompatiblewithreceiverstate:
		return "Message not compatible with receiver state"
	case ies.CauseProtocolSemanticerror:
		return "Semantic error"
	case ies.CauseProtocolAbstractsyntaxerrorfalselyconstructedmessage:
		return "Abstract syntax error - Falsely constructed message"
	case ies.CauseProtocolUnspecified:
		return "Unspecified cause for protocol"
	default:
		return "Unknown cause for protocol"
	}
}

func causeMiscToString(misc *ies.CauseMisc) string {
	switch misc.Value {
	case ies.CauseMiscControlprocessingoverload:
		return "Control processing overload"
	case ies.CauseMiscNotenoughuserplaneprocessingresources:
		return "Not enough user plane processing resources"
	case ies.CauseMiscHardwarefailure:
		return "Hardware failure"
	case ies.CauseMiscOmintervention:
		return "OM (Operations and Maintenance) intervention"
	case ies.CauseMiscUnknownplmn:
		return "Unknown PLMN (Public Land Mobile Network)"
	case ies.CauseMiscUnspecified:
		return "Unspecified cause for miscellaneous"
	default:
		return "Unknown cause for miscellaneous"
	}
}
