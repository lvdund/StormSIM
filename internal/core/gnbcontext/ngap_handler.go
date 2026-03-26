package gnbcontext

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"
	"sync/atomic"

	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
)

func (gnb *GnbContext) getUeFromContext(ranUeId int64, amfUeId int64) *GnbUeContext {
	// check RanUeId and get UE.
	ue, err := gnb.getGnbUe(ranUeId)
	if err != nil || ue == nil {
		gnb.Error("RAN UE NGAP ID is incorrect, found: %d", ranUeId)
		return nil
		// TODO SEND ERROR INDICATION
	}

	ue.amfUeNgapId = amfUeId

	return ue
}

func (gnb *GnbContext) handlerDownlinkNasTransport(msg *ies.DownlinkNASTransport) {
	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID
	nasPayload := msg.NASPDU

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error(
			"Unable to forward Downlink NAS: UE not found with RAN UE ID %d",
			ranUeId)
		return
	}

	// send NAS message to UE.
	ue.sendNasToUe(nasPayload)
}

func (gnb *GnbContext) handlerInitialContextSetupRequest(msg *ies.InitialContextSetupRequest) {

	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID
	nasPayload := msg.NASPDU
	var allowednssai []model.Snssai
	var mobilityRestrict = "not informed"
	var maskedImeisv string
	var ueSecCaps ies.UESecurityCapabilities
	var pDUSessionResourceSetupListCxtReq []ies.PDUSessionResourceSetupItemCxtReq

	// var securityKey []byte
	//TODO: using for create new security context between GNB and UE.
	// securityKey = msg.SecurityKey.Value.Bytes

	allowednssai = make([]model.Snssai, len(msg.AllowedNSSAI))

	// list S-NSSAI(Single - Network Slice Selection Assistance Information).
	for i, items := range msg.AllowedNSSAI {
		allowednssai[i] = model.Snssai{}

		if items.SNSSAI.SST != nil {
			allowednssai[i].Sst = fmt.Sprintf("%x", items.SNSSAI.SST)
		} else {
			allowednssai[i].Sst = "not informed"
		}

		if items.SNSSAI.SD != nil {
			allowednssai[i].Sd = fmt.Sprintf("%x", items.SNSSAI.SD)
		} else {
			allowednssai[i].Sd = "not informed"
		}
	}

	// that field is not mandatory.
	if msg.MobilityRestrictionList == nil {
		gnb.Info("Mobility Restriction is missing")
		mobilityRestrict = "not informed"
	} else {
		mobilityRestrict = fmt.Sprintf("%x", msg.MobilityRestrictionList.ServingPLMN)
	}

	// that field is not mandatory.
	// TODO using for mapping UE context
	if msg.MaskedIMEISV == nil {
		gnb.Info("Masked IMEISV is missing")
		maskedImeisv = "not informed"
	} else {
		maskedImeisv = fmt.Sprintf("%x", msg.MaskedIMEISV)
	}

	// TODO using for create new security context between UE and GNB.
	// TODO algorithms for create new security context between UE and GNB.
	ueSecCaps = msg.UESecurityCapabilities

	if msg.PDUSessionResourceSetupListCxtReq == nil {
		gnb.Info("PDUSessionResourceSetupListCxtReq is missing")
	}
	pDUSessionResourceSetupListCxtReq = msg.PDUSessionResourceSetupListCxtReq

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error("Cannot setup context for unknown UE	with RANUEID %d", ranUeId)
		return
	}
	// create UE context.
	ue.createUeContext(mobilityRestrict, maskedImeisv, allowednssai, &ueSecCaps)

	// show UE context.
	gnb.Info(" UE context created successfully")
	gnb.Info(" RAN ID %d", ue.ranUeNgapId)
	gnb.Info(" AMF ID %d", ue.amfUeNgapId)
	gnb.Info(" Mobility Restrict --Plmn-- Mcc:%s Mnc:%s",
		ue.context.MobilityInfo.Mcc, ue.context.MobilityInfo.Mnc)
	gnb.Info(" Masked Imeisv: %s", ue.context.MaskedIMEISV)
	gnb.Info(" lowed Nssai (Sst-Sd): %v", allowednssai)

	if nasPayload != nil {
		ue.sendNasToUe(nasPayload)
	}

	if pDUSessionResourceSetupListCxtReq != nil {
		gnb.Info("AMF is requesting some PDU Session to be setup during Initial Context Setup")
		for _, pDUSessionResourceSetupItemCtxReq := range pDUSessionResourceSetupListCxtReq {
			pduSessionId := pDUSessionResourceSetupItemCtxReq.PDUSessionID
			sst := fmt.Sprintf("%x", pDUSessionResourceSetupItemCtxReq.SNSSAI.SST)
			sd := "not informed"
			if pDUSessionResourceSetupItemCtxReq.SNSSAI.SD != nil {
				sd = fmt.Sprintf("%x", pDUSessionResourceSetupItemCtxReq.SNSSAI.SD)
			}

			pDUSessionResourceSetupRequestTransferBytes := pDUSessionResourceSetupItemCtxReq.PDUSessionResourceSetupRequestTransfer
			pDUSessionResourceSetupRequestTransfer := &ies.PDUSessionResourceSetupRequestTransfer{}
			if err, _ := pDUSessionResourceSetupRequestTransfer.Decode(pDUSessionResourceSetupRequestTransferBytes); err != nil {
				gnb.Error("Unable to unmarshall PDUSessionResourceSetupRequestTransfer: %s", err.Error())
				continue
			}

			var gtpTunnel *ies.GTPTunnel
			var upfIp string
			var teidUplink aper.OctetString

			if pDUSessionResourceSetupRequestTransfer.ULNGUUPTNLInformation.GTPTunnel != nil {
				gtpTunnel = pDUSessionResourceSetupRequestTransfer.ULNGUUPTNLInformation.GTPTunnel
				upfIp, _ = utils.IPAddressToString(gtpTunnel.TransportLayerAddress)
				teidUplink = gtpTunnel.GTPTEID
			}

			if _, err := ue.createPduSession(int64(pduSessionId), upfIp, sst, sd, 0, 1, 0, 0, binary.BigEndian.Uint32(teidUplink), gnb.getUeTeid(ue)); err != nil {
				gnb.Error("", err)
			}

			if pDUSessionResourceSetupItemCtxReq.NASPDU != nil {
				ue.sendNasToUe(pDUSessionResourceSetupItemCtxReq.NASPDU)
			}
		}

		ue.sendMsgToUe(&model.RlinkSetupPduSessonCommand{
			PrUeId:         ue.prUeId,
			GNBPduSessions: ue.context.PduSession,
			GnbIp:          gnb.dataPlaneInfo.gnbIp,
		})
	}

	// send Initial Context Setup Response.
	gnb.Info("Send Initial Context Setup Response.")
	gnb.sendInitialContextSetupResponse(ue)
}

func (gnb *GnbContext) handlerPduSessionResourceSetupRequest(msg *ies.PDUSessionResourceSetupRequest) {

	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID
	var pDUSessionResourceSetupList []ies.PDUSessionResourceSetupItemSUReq

	// TODO MORE FIELDS TO CHECK HERE
	if msg.PDUSessionResourceSetupListSUReq == nil {
		gnb.Fatal("PDU SESSION RESOURCE SETUP LIST SU REQ is missing")
	} else {
		pDUSessionResourceSetupList = msg.PDUSessionResourceSetupListSUReq
	}

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error("Cannot setup PDU Session for unknown UE With RANUEID %d", ranUeId)
		return
	}

	var configuredPduSessions []*model.GnbPDUSessionContext
	for _, item := range pDUSessionResourceSetupList {
		var pduSessionId int64
		var ulTeid uint32
		var upfAddress []byte
		var nasPayload []byte
		var sst string
		var sd string
		var pduSessionType uint64
		var qosId int64
		var fiveQi int64
		var priArp int64

		// check PDU Session NAS PDU.
		if item.PDUSessionNASPDU != nil {
			nasPayload = item.PDUSessionNASPDU
		} else {
			gnb.Fatal("NAS PDU is missing")
		}

		// check pdu session id and nssai information for create a PDU Session.

		// create a PDU session(PDU SESSION ID + NSSAI).
		pduSessionId = int64(item.PDUSessionID)

		if item.SNSSAI.SD != nil {
			sd = fmt.Sprintf("%x", item.SNSSAI.SD)
		} else {
			sd = "not informed"
		}

		if item.SNSSAI.SST != nil {
			sst = fmt.Sprintf("%x", item.SNSSAI.SST)
		} else {
			sst = "not informed"
		}

		if item.PDUSessionResourceSetupRequestTransfer != nil {

			pDUSessionResourceSetupRequestTransfer := &ies.PDUSessionResourceSetupRequestTransfer{}
			if err, _ := pDUSessionResourceSetupRequestTransfer.Decode(item.PDUSessionResourceSetupRequestTransfer); err == nil {

				ulTeid = binary.BigEndian.Uint32(pDUSessionResourceSetupRequestTransfer.ULNGUUPTNLInformation.GTPTunnel.GTPTEID)
				upfAddress = pDUSessionResourceSetupRequestTransfer.ULNGUUPTNLInformation.GTPTunnel.TransportLayerAddress.Bytes

				for _, itemsQos := range pDUSessionResourceSetupRequestTransfer.QosFlowSetupRequestList {
					qosId = itemsQos.QosFlowIdentifier
					fiveQi = itemsQos.QosFlowLevelQosParameters.QosCharacteristics.NonDynamic5QI.FiveQI
					priArp = itemsQos.QosFlowLevelQosParameters.AllocationAndRetentionPriority.PriorityLevelARP
				}

				pduSessionType = uint64(pDUSessionResourceSetupRequestTransfer.PDUSessionType.Value)
			} else {
				gnb.Info("Error in decode Pdu Session Resource Setup Request Transfer")
			}
		} else {
			gnb.Fatal("Error in Pdu Session Resource Setup Request, Pdu Session Resource Setup Request Transfer is missing")
		}

		upfIp := fmt.Sprintf("%d.%d.%d.%d", upfAddress[0], upfAddress[1], upfAddress[2], upfAddress[3])

		// create PDU Session for GNB UE.
		pduSession, err := ue.createPduSession(pduSessionId, upfIp, sst, sd, pduSessionType, qosId, priArp, fiveQi, ulTeid, gnb.getUeTeid(ue))
		if err != nil {
			gnb.Error("Error in Pdu Session Resource Setup Request: %v", err)
		}
		configuredPduSessions = append(configuredPduSessions, pduSession)

		gnb.Info("PDU session established successfully")
		gnb.Info("PDU Session Id: %d", pduSession.PduSessionId)

		sst, sd = ue.getSelectedNssai(pduSession.PduSessionId)
		gnb.Info("PDU Session Id: %d - Type: %s", pduSession.PduSessionId, pduSession.GetPduType())
		gnb.Info("\tNSSAI Selected --- sst:%s sd:%s", sst, sd)
		gnb.Info("\tQoS Flow Identifier: %d", pduSession.QosId)
		gnb.Info("\tUplink Teid: %d", pduSession.UplinkTeid)
		gnb.Info("\tDownlink Teid: %d", pduSession.DownlinkTeid)
		gnb.Info("\tNon-Dynamic-5QI: %d", pduSession.FiveQi)
		gnb.Info("\tPriority Level ARP: %d", pduSession.PriArp)
		gnb.Info("\tUPF Address: %s:2152", fmt.Sprintf("%d.%d.%d.%d", upfAddress[0], upfAddress[1], upfAddress[2], upfAddress[3]))

		// send NAS message to UE.
		ue.sendNasToUe(nasPayload)

		var pduSessions [16]*model.GnbPDUSessionContext
		pduSessions[0] = pduSession

		ue.sendMsgToUe(&model.RlinkSetupPduSessonCommand{
			GnbIp:          gnb.dataPlaneInfo.gnbIp,
			GNBPduSessions: pduSessions,
		})
	}

	// send PDU Session Resource Setup Response.
	gnb.sendPduSessionResourceSetupResponse(configuredPduSessions, ue)
}

func (gnb *GnbContext) handlerPduSessionReleaseCommand(msg *ies.PDUSessionResourceReleaseCommand) {
	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID
	nasPayload := msg.NASPDU
	var pduSessionIds []int64

	//TODO: MORE FIELDS TO CHECK HERE
	if msg.PDUSessionResourceToReleaseListRelCmd == nil {
		gnb.Fatal("PDU SESSION RESOURCE SETUP LIST SU REQ is missing")
	} else {
		for _, pDUSessionRessourceToReleaseItemRelCmd := range msg.PDUSessionResourceToReleaseListRelCmd {
			pduSessionIds = append(pduSessionIds, pDUSessionRessourceToReleaseItemRelCmd.PDUSessionID)
		}
	}

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error("Cannot release PDU Session for unknown UE With RANUEID %d", ranUeId)
		return
	}

	for _, pduSessionId := range pduSessionIds {
		pduSession, err := ue.getPduSession(pduSessionId)
		if pduSession == nil || err != nil {
			gnb.Error("Unable to delete PDU Session ", pduSessionId, " from UE as the PDU Session was not found. Ignoring.")
			continue
		}
		ue.deletePduSession(pduSessionId)
		gnb.Info("Successfully deleted PDU Session ", pduSessionId, " from UE Context")
	}

	gnb.sendPduSessionReleaseResponse(pduSessionIds, ue)

	ue.sendNasToUe(nasPayload)
}

func (gnb *GnbContext) handlerNgSetupResponse(amf *GnbAmfContext, msg *ies.NGSetupResponse) {
	gnb.Info("Receive NGSetupResponse")
	err := false
	var plmn string

	// information about AMF and add in AMF context.
	amfName := msg.AMFName
	amf.name = string(amfName)

	amf.amfCapacityValue = msg.RelativeAMFCapacity

	if msg.PLMNSupportList == nil {
		gnb.Info("In NG SETUP RESPONSE, PLMN Support list is missing")
		err = true
	}

	for _, items := range msg.PLMNSupportList {

		plmn = fmt.Sprintf("%x", items.PLMNIdentity)
		amf.addedPlmn(plmn)

		if items.SliceSupportList == nil {
			gnb.Info("Error in NG SETUP RESPONSE, PLMN Support list is inappropriate")
			gnb.Info("Error in NG SETUP RESPONSE, Slice Support list is missing")
			err = true
		}

		for _, slice := range items.SliceSupportList {

			var sd string
			var sst string

			if slice.SNSSAI.SST != nil {
				sst = fmt.Sprintf("%x", slice.SNSSAI.SST)
			} else {
				sst = "was not informed"
			}

			if slice.SNSSAI.SD != nil {
				sd = fmt.Sprintf("%x", slice.SNSSAI.SD)
			} else {
				sd = "was not informed"
			}

			// update amf slice supported
			amf.addedSlice(sst, sd)
		}
	}

	if err {
		gnb.Fatal("AMF is inactive")
		amf.state = AMF_INACTIVE
	} else {
		amf.state = AMF_ACTIVE
		gnb.Info("AMF Name: %s - state: Active - capacity: %d", amf.name, amf.amfCapacityValue)
		for i := range amf.lenPlmn {
			mcc, mnc := amf.getPlmnSupport(i)
			gnb.Info("\tPLMNs Identities Supported by AMF -- mcc:%s mnc:%s", mcc, mnc)
		}
		for i := range amf.lenSlice {
			sst, sd := amf.getSliceSupport(i)
			gnb.Info("\tList of AMF slices Supported by AMF -- sst:%s sd:%s", sst, sd)
		}
		gnb.isReady <- true
	}
}

func (gnb *GnbContext) handlerUeContextReleaseCommand(msg *ies.UEContextReleaseCommand) {

	cause := msg.Cause
	ranue_id := msg.UENGAPIDs.UENGAPIDpair.RANUENGAPID

	ue, err := gnb.getGnbUe(ranue_id)
	if err != nil {
		gnb.Error("AMF is trying to free the context of an unknown UE")
		return
	}

	// Send UEContextReleaseComplete
	gnb.sendUeContextReleaseComplete(ue)

	gnb.Info("Releasing UE Context, cause: %s", causeToString(&cause))
}

var CountHO int32

func (gnb *GnbContext) handlerPathSwitchRequestAcknowledge(msg *ies.PathSwitchRequestAcknowledge) {
	var pduSessionResourceSwitchedList []ies.PDUSessionResourceSwitchedItem

	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID

	pduSessionResourceSwitchedList = msg.PDUSessionResourceSwitchedList
	if pduSessionResourceSwitchedList == nil {
		gnb.Fatal("PduSessionResourceSwitchedList is missing")
		gnb.Warn("No PDU Sessions to be switched")
		// TODO SEND ERROR INDICATION
		return
	}

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error("Cannot Xn Handover unknown UE With RANUEID %d", ranUeId)
		return
	}

	for _, pduSessionResourceSwitchedItem := range pduSessionResourceSwitchedList {
		pduSessionId := pduSessionResourceSwitchedItem.PDUSessionID
		pduSession, err := ue.getPduSession(pduSessionId)
		if err != nil {
			gnb.Error("Trying to path switch an unknown PDU Session ID ", pduSessionId, ": ", err)
			continue
		}

		pathSwitchRequestAcknowledgeTransferBytes := pduSessionResourceSwitchedItem.PathSwitchRequestAcknowledgeTransfer
		pathSwitchRequestAcknowledgeTransfer := &ies.PathSwitchRequestAcknowledgeTransfer{}
		err = pathSwitchRequestAcknowledgeTransfer.Decode(pathSwitchRequestAcknowledgeTransferBytes)
		if err != nil {
			gnb.Error("Unable to unmarshall PathSwitchRequestAcknowledgeTransfer: ", err)
			continue
		}

		if pathSwitchRequestAcknowledgeTransfer.ULNGUUPTNLInformation != nil {
			gtpTunnel := pathSwitchRequestAcknowledgeTransfer.ULNGUUPTNLInformation.GTPTunnel
			upfIpv4, _ := utils.IPAddressToString(gtpTunnel.TransportLayerAddress)
			teidUplink := gtpTunnel.GTPTEID

			// Set new Teid Uplink received in PathSwitchRequestAcknowledge
			pduSession.UplinkTeid = binary.BigEndian.Uint32(teidUplink)
			pduSession.UpfIp = upfIpv4
		}
		var pduSessions [16]*model.GnbPDUSessionContext
		pduSessions[0] = pduSession

		ue.sendMsgToUe(&model.RlinkSetupPduSessonCommand{
			GNBPduSessions: pduSessions,
			GnbIp:          gnb.dataPlaneInfo.gnbIp,
		})
	}

	if _, ok := gnb.ueHoStatusPool.Load(ue.prUeId); ok {
		gnb.ueHoStatusPool.Store(ue.prUeId, true)
	}
	atomic.AndInt32(&CountHO, 1)
	gnb.Info("Handover completed successfully for UE-ranNgapId-%d: %d", ue.ranUeNgapId, CountHO)
}

func (gnb *GnbContext) handlerHandoverRequest(amf *GnbAmfContext, msg *ies.HandoverRequest) {
	_ = amf

	var allowednssai []model.Snssai
	var maskedImeisv string

	amfUeId := msg.AMFUENGAPID

	ueSecCaps := msg.UESecurityCapabilities

	if msg.AllowedNSSAI == nil {
		gnb.Fatal("Allowed NSSAI is missing")
	} else {
		allowednssai = make([]model.Snssai, len(msg.AllowedNSSAI))

		// list S-NSSAI(Single - Network Slice Selection Assistance Information).
		for i, items := range msg.AllowedNSSAI {
			allowednssai[i] = model.Snssai{}

			if items.SNSSAI.SST != nil {
				allowednssai[i].Sst = fmt.Sprintf("%x", items.SNSSAI.SST)
			} else {
				allowednssai[i].Sst = "not informed"
			}

			if items.SNSSAI.SD != nil {
				allowednssai[i].Sd = fmt.Sprintf("%x", items.SNSSAI.SD)
			} else {
				allowednssai[i].Sd = "not informed"
			}
		}
	}

	// that field is not mandatory.
	// TODO using for mapping UE context
	if msg.MaskedIMEISV == nil {
		gnb.Info("Masked IMEISV is missing")
		maskedImeisv = "not informed"
	} else {
		maskedImeisv = fmt.Sprintf("%x", msg.MaskedIMEISV)
	}

	pDUSessionResourceSetupListHOReq := msg.PDUSessionResourceSetupListHOReq
	if pDUSessionResourceSetupListHOReq == nil {
		gnb.Fatal("pDUSessionResourceSetupListHOReq is missing")
		// TODO SEND ERROR INDICATION
	}

	sourceToTargetContainer := msg.SourceToTargetTransparentContainer
	if sourceToTargetContainer == nil {
		gnb.Error("HandoverRequest message from AMF is missing mandatory SourceToTargetTransparentContainer")
		return
	}

	sourceToTargetContainerNgap := &ies.SourceNGRANNodeToTargetNGRANNodeTransparentContainer{}
	err := sourceToTargetContainerNgap.Decode(sourceToTargetContainer)
	if err != nil {
		gnb.Error("Unable to unmarshall SourceToTargetTransparentContainer: ", err)
		return
	}
	if sourceToTargetContainerNgap.IndexToRFSP == nil {
		gnb.Error("SourceToTargetTransparentContainer from source gNodeB is missing IndexToRFSP")
		return
	}
	prUeId := sourceToTargetContainerNgap.IndexToRFSP

	ue, err := gnb.newGnBUe(nil, *prUeId, "", nil)
	if ue == nil || err != nil {
		gnb.Error("HandoverFailure: %s", err)
	}
	ue.amfUeNgapId = amfUeId

	ue.createUeContext("not informed", maskedImeisv, allowednssai, &ueSecCaps)

	for _, pDUSessionResourceSetupItemHOReq := range pDUSessionResourceSetupListHOReq {
		pduSessionId := pDUSessionResourceSetupItemHOReq.PDUSessionID
		sst := fmt.Sprintf("%x", pDUSessionResourceSetupItemHOReq.SNSSAI.SST)
		sd := "not informed"
		if pDUSessionResourceSetupItemHOReq.SNSSAI.SD != nil {
			sd = fmt.Sprintf("%x", pDUSessionResourceSetupItemHOReq.SNSSAI.SD)
		}

		handOverRequestTransferBytes := pDUSessionResourceSetupItemHOReq.HandoverRequestTransfer
		handOverRequestTransfer := &ies.PDUSessionResourceSetupRequestTransfer{}
		if err, _ := handOverRequestTransfer.Decode(handOverRequestTransferBytes); err != nil {
			gnb.Error("Unable to unmarshall HandOverRequestTransfer: ", err)
			continue
		}

		var gtpTunnel *ies.GTPTunnel

		gtpTunnel = handOverRequestTransfer.ULNGUUPTNLInformation.GTPTunnel
		upfIp, _ := utils.IPAddressToString(gtpTunnel.TransportLayerAddress)
		teidUplink := gtpTunnel.GTPTEID

		_, err = ue.createPduSession(int64(pduSessionId), upfIp, sst, sd, 0, 1, 0, 0, binary.BigEndian.Uint32(teidUplink), gnb.getUeTeid(ue))
		if err != nil {
			gnb.Error("", err)
		}
	}

	gnb.sendHandoverRequestAcknowledge(ue)
}

func (gnb *GnbContext) handlerHandoverCommand(amf *GnbAmfContext, msg *ies.HandoverCommand) {
	_ = amf

	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID

	ue := gnb.getUeFromContext(ranUeId, amfUeId)
	if ue == nil {
		gnb.Error("Cannot NGAP  Handover unknown UE With RANUEID %d", ranUeId)
		return
	}
	newGnb := ue.handoverTargetGnb
	if newGnb == nil {
		gnb.Error("AMF is sending a Handover Command for an UE we did not send a Handover Required message")
		// TODO SEND ERROR INDICATION
		return
	}

	conn := rlink.NewConnection(
		ue.prUeId,
		ue.msin,
		newGnb.controlPlaneInfo.gnbId,
		rlink.DefaultBufferSize,
		rlink.DefaultDuration,
	)
	SendToGnb(gnb.controlPlaneInfo.gnbId, &model.RLinkHandoverForwardUeContext{
		PrUeId:      ue.prUeId,
		Msin:        ue.msin,
		Conn:        conn,
		SourceGnbId: gnb.controlPlaneInfo.gnbId,
	}, newGnb.controlPlaneInfo.gnbId, true)

	ue.sendMsgToUe(&model.RLinkHandoverPrepareRequest{
		PrUeId:       ue.prUeId,
		Conn:         conn,
		TargetGnbId:  newGnb.controlPlaneInfo.gnbId,
		IsN2Handover: true,
	})
}

func (gnb *GnbContext) handlerPaging(msg *ies.Paging) {

	uEPagingIdentity := msg.UEPagingIdentity
	var tAIListForPaging []ies.TAIListForPagingItem

	if msg.TAIListForPaging == nil {
		gnb.Fatal("TAI List For Paging is missing")
	} else {
		tAIListForPaging = msg.TAIListForPaging
	}

	_ = tAIListForPaging

	gnb.addPagedUE(uEPagingIdentity.FiveGSTMSI)

	gnb.Info("Paging UE")
}

func (gnb *GnbContext) handlerErrorIndication(msg *ies.ErrorIndication) {

	ranUeId := msg.RANUENGAPID
	amfUeId := msg.AMFUENGAPID
	if ranUeId == nil {
		gnb.Error("Received an Error Indication: missing ran UE id")
		return
	}
	if amfUeId == nil {
		gnb.Error("Received an Error Indication: missing amf UE id")
		return
	}

	ue, err := gnb.getGnbUe(*ranUeId)
	if err != nil {
		gnb.Error("Received an Error Indication for UE with AMF UE ID: %d, RAN UE ID: %d with err %s", *amfUeId, *ranUeId, err.Error())
		return
	}

	gnb.Error("Received an Error Indication for UE %s with AMF UE ID: %d, RAN UE ID: %d", ue.msin, *amfUeId, *ranUeId)
}

func (gnb *GnbContext) handlerAmfConfigurationUpdate(amf *GnbAmfContext, msg *ies.AMFConfigurationUpdate) {
	gnb.Debug("Before Update:")

	amfPool := &gnb.amfPool
	amfPool.Range(func(k, v any) bool {
		oldAmf, ok := v.(*GnbAmfContext)
		if ok {
			tnla := oldAmf.tlnaAssoc
			gnb.Debug("[AMF Name: %5s], IP: %10s, AMFCapacity: %3d, TNLA Weight Factor: %2d, TNLA Usage: %2d\n",
				oldAmf.name, oldAmf.amfIp, oldAmf.amfCapacityValue, tnla.weightFactor, tnla.usage)
		}
		return true
	})

	var amfCapacity int64
	var amfRegionId, amfSetId, amfPointer []byte

	amfName := string(msg.AMFName)

	if msg.ServedGUAMIList != nil {
		for _, servedGuamiItem := range msg.ServedGUAMIList {
			amfRegionId = servedGuamiItem.GUAMI.AMFRegionID.Bytes
			amfSetId = servedGuamiItem.GUAMI.AMFSetID.Bytes
			amfPointer = servedGuamiItem.GUAMI.AMFPointer.Bytes
		}
	}

	if msg.RelativeAMFCapacity != nil {
		amfCapacity = *msg.RelativeAMFCapacity
	}

	if msg.AMFTNLAssociationToAddList != nil {
		toAddList := msg.AMFTNLAssociationToAddList
		for _, toAddItem := range toAddList {
			bitLen := len(toAddItem.AMFTNLAssociationAddress.EndpointIPAddress.Bytes) * 8
			var ipv4String string
			if bitLen == 32 || bitLen == 160 { // IPv4 or IPv4+IPv6
				ipv4String, _ = utils.IPAddressToString(*toAddItem.AMFTNLAssociationAddress.EndpointIPAddress)
			}

			amfPool := &gnb.amfPool
			amfExisted := false
			amfPool.Range(func(key, value any) bool {
				gnbAmf, ok := value.(*GnbAmfContext)
				if !ok {
					return true
				}
				if gnbAmf.amfIp == ipv4String {
					gnb.Info("SCTP/NGAP service exists")
					amfExisted = true
					return false
				}
				return true
			})
			if amfExisted {
				continue
			}

			port := 38412 // default sctp port
			newAmf := gnb.newGnbAmf(ipv4String, port)
			newAmf.name = amfName
			newAmf.amfCapacityValue = amfCapacity
			newAmf.setRegionId(amfRegionId)
			newAmf.setSetId(amfSetId)
			newAmf.setPointer(amfPointer)
			newAmf.tlnaAssoc.usage = toAddItem.TNLAssociationUsage.Value
			newAmf.tlnaAssoc.weightFactor = toAddItem.TNLAddressWeightFactor

			// start communication with AMF(SCTP).
			if err := gnb.initConn(newAmf); err != nil {
				gnb.Fatal("Error in", err)
			} else {
				gnb.Info("SCTP/NGAP service is running")
				// wg.Add(1)
			}

			gnb.sendNgSetupRequest(newAmf)

		}
	}

	if msg.AMFTNLAssociationToRemoveList != nil {
		toRemoveList := msg.AMFTNLAssociationToRemoveList
		for _, toRemoveItem := range toRemoveList {
			bitLen := toRemoveItem.AMFTNLAssociationAddress.EndpointIPAddress.NumBits
			var ipv4String string
			if bitLen == 32 || bitLen == 160 { // IPv4 or IPv4+IPv6
				ipv4String, _ = utils.IPAddressToString(*toRemoveItem.AMFTNLAssociationAddress.EndpointIPAddress)
			}
			port := 38412 // default sctp port
			amfPool := &gnb.amfPool
			amfPool.Range(func(k, v any) bool {
				oldAmf, ok := v.(*GnbAmfContext)
				if ok && oldAmf.amfIp == ipv4String && oldAmf.amfPort == port {
					gnb.Info("Remove AMF:", amf.name, " IP:", amf.amfIp)
					amf.tlnaAssoc.sctpConn.Close() // Close SCTP Conntection
					amfPool.Delete(k)
					return false
				}
				return true
			})
		}
	}

	if msg.AMFTNLAssociationToUpdateList != nil {
		toUpdateList := msg.AMFTNLAssociationToUpdateList
		for _, toUpdateItem := range toUpdateList {
			bitLen := toUpdateItem.AMFTNLAssociationAddress.EndpointIPAddress.NumBits
			var ipv4String string
			if bitLen == 32 || bitLen == 160 {
				ipv4String, _ = utils.IPAddressToString(*toUpdateItem.AMFTNLAssociationAddress.EndpointIPAddress)
			}
			port := 38412 // default sctp port
			amfPool := &gnb.amfPool
			amfPool.Range(func(k, v any) bool {
				oldAmf, ok := v.(*GnbAmfContext)
				if ok && oldAmf.amfIp == ipv4String && oldAmf.amfPort == port {
					oldAmf.name = amfName
					oldAmf.amfCapacityValue = amfCapacity
					oldAmf.setRegionId(amfRegionId)
					oldAmf.setSetId(amfSetId)
					oldAmf.setPointer(amfPointer)

					oldAmf.tlnaAssoc.usage = toUpdateItem.TNLAssociationUsage.Value
					oldAmf.tlnaAssoc.weightFactor = *toUpdateItem.TNLAddressWeightFactor
					return false
				}
				return true
			})
		}
	}

	gnb.Debug("After Update:")
	amfPool = &gnb.amfPool
	amfPool.Range(func(k, v any) bool {
		oldAmf, ok := v.(*GnbAmfContext)
		if ok {
			tnla := oldAmf.tlnaAssoc
			gnb.Debug("[AMF Name: %5s], IP: %10s, AMFCapacity: %3d, TNLA Weight Factor: %2d, TNLA Usage: %2d\n",
				oldAmf.name, oldAmf.amfIp, oldAmf.amfCapacityValue, tnla.weightFactor, tnla.usage)
		}
		return true
	})

	gnb.sendAmfConfigurationUpdateAcknowledge(amf)
}

func (gnb *GnbContext) handlerAmfStatusIndication(amf *GnbAmfContext, msg *ies.AMFStatusIndication) {
	_ = amf

	if msg.UnavailableGUAMIList != nil {
		for _, unavailableGuamiItem := range msg.UnavailableGUAMIList {
			octetStr := unavailableGuamiItem.GUAMI.PLMNIdentity
			hexStr := fmt.Sprintf("%02x%02x%02x", octetStr[0], octetStr[1], octetStr[2])
			var unavailableMcc, unavailableMnc string
			unavailableMcc = string(hexStr[1]) + string(hexStr[0]) + string(hexStr[3])
			unavailableMnc = string(hexStr[5]) + string(hexStr[4])
			if hexStr[2] != 'f' {
				unavailableMnc = string(hexStr[2]) + string(hexStr[5]) + string(hexStr[4])
			}

			amfPool := &gnb.amfPool

			// select backup AMF
			var backupAmf *GnbAmfContext
			amfPool.Range(func(k, v any) bool {
				amf, ok := v.(*GnbAmfContext)
				if !ok {
					return true
				}
				if unavailableGuamiItem.BackupAMFName != nil &&
					amf.name == string(unavailableGuamiItem.BackupAMFName) {
					backupAmf = amf
					return false
				}

				return true
			})

			if backupAmf == nil {
				return
			}

			amfPool.Range(func(k, v any) bool {
				oldAmf, ok := v.(*GnbAmfContext)
				if !ok {
					return true
				}
				for j := range oldAmf.lenPlmn {
					oldAmfSupportMcc, oldAmfSupportMnc := oldAmf.getPlmnSupport(j)

					if oldAmfSupportMcc == unavailableMcc && oldAmfSupportMnc == unavailableMnc &&
						reflect.DeepEqual(oldAmf.regionId, unavailableGuamiItem.GUAMI.AMFRegionID) &&
						reflect.DeepEqual(oldAmf.setId, unavailableGuamiItem.GUAMI.AMFSetID) &&
						reflect.DeepEqual(oldAmf.amfPointer, unavailableGuamiItem.GUAMI.AMFPointer) {

						gnb.Info("Remove AMF: [", "Id: ", oldAmf.amfId,
							"Name: ", oldAmf.name, "Ipv4: ", oldAmf.amfIp, "]")

						// NGAP UE-TNLA Rebinding
						uePool := &gnb.ranUePool
						uePool.Range(func(k, v any) bool {
							ue, ok := v.(*GnbUeContext)
							if !ok {
								return true
							}

							if ue.amfId == oldAmf.amfId {
								// set amfId and SCTP association for UE.
								ue.amfId = backupAmf.amfId
								ue.sctpConnection = backupAmf.tlnaAssoc.sctpConn
							}

							return true
						})

						prUePool := &gnb.prUeIdPool
						prUePool.Range(func(k, v any) bool {
							ue, ok := v.(*GnbUeContext)
							if !ok {
								return true
							}

							if ue.amfId == oldAmf.amfId {
								// set amfId and SCTP association for UE.
								ue.amfId = backupAmf.amfId
								ue.sctpConnection = backupAmf.tlnaAssoc.sctpConn
							}

							return true
						})

						oldAmf.tlnaAssoc.sctpConn.Close()
						amfPool.Delete(k)

						return true
					}
				}
				return true
			})
		}
	}
}
