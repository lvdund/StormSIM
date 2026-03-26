package gnbcontext

import (
	"bytes"
	"encoding/binary"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/model"

	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
)

func TriggerReleaseUe(gnb *GnbContext, msin string) error {
	ue, err := gnb.getGnbUeByMsin(msin)
	if err != nil || ue == nil {
		gnb.Error("Cannot trigger release ue %s: %s", msin, err.Error())
		return err
	}
	gnb.sendUeContextReleaseRequest(ue, &ies.Cause{
		Choice: ies.CausePresentRadionetwork, RadioNetwork: &ies.CauseRadioNetwork{
			Value: ies.CauseRadioNetworkUnspecified,
		},
	})

	return nil
}

func (gnb *GnbContext) GeListUeInfo() []*GnbUeContext {
	return nil
}

func (gnb *GnbContext) sendHandoverNotify(ue *GnbUeContext) {
	gnb.Info("Initiating Handover Notify")
	PLMNIdentity := gnb.getPLMNIdentityInBytes()
	NRCellIdentity := gnb.getNRCellIdentity()
	TAC := gnb.getTacInBytes()

	msg := &ies.HandoverNotify{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					PLMNIdentity:   PLMNIdentity,
					NRCellIdentity: NRCellIdentity,
				},
				TAI: ies.TAI{
					PLMNIdentity: PLMNIdentity,
					TAC:          TAC,
				},
			},
		},
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Handover Notify: ", err)
		return
	}

	gnb.LogNgapSend("HandoverNotify")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Handover Notify: ", err)
	}
}

func (gnb *GnbContext) sendPathSwitchRequest(ue *GnbUeContext) {
	gnb.Info("Initiating Path Switch Request")
	pduSessions := ue.context.PduSession

	msg := &ies.PathSwitchRequest{
		SourceAMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID:       ue.ranUeNgapId,
		UserLocationInformation: ies.UserLocationInformation{
			Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
			UserLocationInformationNR: &ies.UserLocationInformationNR{
				NRCGI: ies.NRCGI{
					PLMNIdentity:   gnb.getPLMNIdentityInBytes(),
					NRCellIdentity: gnb.getNRCellIdentity(),
				},
				TAI: ies.TAI{
					PLMNIdentity: gnb.getPLMNIdentityInBytes(),
					TAC:          gnb.getTacInBytes(),
				},
			},
		},
		UESecurityCapabilities:               *ue.context.UeSecurityCapabilities,
		PDUSessionResourceToBeSwitchedDLList: []ies.PDUSessionResourceToBeSwitchedDLItem{},
	}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		ip := utils.IPAddressToNgap(gnb.dataPlaneInfo.gnbIp, "")
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.BigEndian, pduSession.DownlinkTeid)
		transfer := ies.PathSwitchRequestTransfer{
			DLNGUUPTNLInformation: ies.UPTransportLayerInformation{
				Choice: ies.UPTransportLayerInformationPresentGtptunnel,
				GTPTunnel: &ies.GTPTunnel{
					TransportLayerAddress: ip,
					GTPTEID:               buf.Bytes(),
				}},
			DLNGUTNLInformationReused:    nil,
			UserPlaneSecurityInformation: nil,
			QosFlowAcceptedList: []ies.QosFlowAcceptedItem{
				{QosFlowIdentifier: pduSession.QosId},
			},
		}

		var b []byte
		var err error
		if b, err = transfer.Encode(); err != nil {
			gnb.Info("Error encoding Path Switch Request ", err)
			return
		}
		msg.PDUSessionResourceToBeSwitchedDLList = append(
			msg.PDUSessionResourceToBeSwitchedDLList,
			ies.PDUSessionResourceToBeSwitchedDLItem{
				PDUSessionID:              pduSession.PduSessionId,
				PathSwitchRequestTransfer: b,
			},
		)
	}

	if len(msg.PDUSessionResourceToBeSwitchedDLList) == 0 {
		gnb.Warn("No PDU Session to handover: Xn Handover requires at least 1 PDU Session")
		msg.PDUSessionResourceToBeSwitchedDLList = nil
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Path Switch Request ", err)
		return
	}
	gnb.LogNgapSend("PathSwitchRequest")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Path Switch Request: ", err)
	}
}

func (gnb *GnbContext) sendUeContextReleaseRequest(ue *GnbUeContext, cause *ies.Cause) {
	gnb.Info("Initiating UE Context Release Request")
	msg := &ies.UEContextReleaseRequest{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
		Cause:       *cause,
	}

	activePduSession := []*model.GnbPDUSessionContext{}
	pduSessions := ue.context.PduSession
	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		activePduSession = append(activePduSession, pduSession)
	}

	if len(activePduSession) > 0 {
		msg.PDUSessionResourceListCxtRelReq = make([]ies.PDUSessionResourceItemCxtRelReq, len(activePduSession))

		// PDU Session Resource Item in PDU session Resource List
		for _, pduSessionID := range activePduSession {
			id := pduSessionID.PduSessionId
			msg.PDUSessionResourceListCxtRelReq = append(msg.PDUSessionResourceListCxtRelReq, ies.PDUSessionResourceItemCxtRelReq{
				PDUSessionID: id,
			})
		}
	}

	ngapPdu, err := ngap.NgapEncode(msg)
	if err != nil {
		gnb.Error("Error create UE Context Release Request: %s", err.Error())
		return
	}

	gnb.LogNgapSend("UEContextReleaseRequest")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending UE Context Release Request: %s", err.Error())
	}
}

func (gnb *GnbContext) sendAmfConfigurationUpdateAcknowledge(amf *GnbAmfContext) {
	gnb.Info("Initiating AMF Configuration Update Acknowledge")
	message := ies.AMFConfigurationUpdateAcknowledge{}

	ngapPdu, err := ngap.NgapEncode(&message)
	if err != nil {
		gnb.Warn("Error sending AMF Configuration Update Acknowledge: ", err)
	}

	gnb.LogNgapSend("AMFConfigurationUpdateAcknowledge")
	amf.sendNgap(ngapPdu)
	if err != nil {
		gnb.Warn("Error sending AMF Configuration Update Acknowledge: ", err)
	}
}

func (gnb *GnbContext) sendNgSetupRequest(amf *GnbAmfContext) {
	gnb.Info("Initiating NG Setup Request")

	msg := ies.NGSetupRequest{}

	msg.GlobalRANNodeID = ies.GlobalRANNodeID{
		Choice: ies.GlobalRANNodeIDPresentGlobalgnbId,
		GlobalGNBID: &ies.GlobalGNBID{
			PLMNIdentity: gnb.getPlmnInOctets(),
			GNBID: ies.GNBID{
				Choice: ies.GNBIDPresentGnbId,
				GNBID: &aper.BitString{
					Bytes:   gnb.getGnbIdInBytes(),
					NumBits: 24,
				},
			},
		},
	}

	msg.RANNodeName = []byte("StormSim")

	sst, sd := gnb.getSliceInBytes()
	msg.SupportedTAList = []ies.SupportedTAItem{
		{
			TAC: gnb.getTacInBytes(),
			BroadcastPLMNList: []ies.BroadcastPLMNItem{
				{
					PLMNIdentity: gnb.getPlmnInOctets(),
					TAISliceSupportList: []ies.SliceSupportItem{
						{SNSSAI: ies.SNSSAI{SST: sst, SD: sd}},
					},
				},
			},
		},
	}

	msg.DefaultPagingDRX = ies.PagingDRX{Value: ies.PagingDRXV128}

	ngapPdu, err := ngap.NgapEncode(&msg)

	if err != nil {
		gnb.Error("Error sending NG Setup Request: ", err)
	}

	gnb.LogNgapSend("NGSetupRequest")
	amf.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending NG Setup Request: ", err)
	}

}

func TriggerXnHandover(oldGnb *GnbContext, newGnb *GnbContext, prUeId int64) {
	newGnb.Info("Initiating Xn UE Handover")

	gnbUeContext, err := oldGnb.getGnbUeByPrUeId(prUeId)
	if err != nil {
		newGnb.Error("Error getting UE from PR UE ID: %s", err.Error())
		return
	}

	conn := rlink.NewConnection(gnbUeContext.prUeId, gnbUeContext.msin,
		newGnb.controlPlaneInfo.gnbId, rlink.DefaultBufferSize, rlink.DefaultDuration)

	// forward ue context, conn to newGnb
	SendToGnb(oldGnb.GetId(), &model.RLinkHandoverForwardUeContext{
		PrUeId:        gnbUeContext.prUeId,
		Msin:          gnbUeContext.msin,
		Conn:          conn,
		AmfUeNgapId:   gnbUeContext.amfUeNgapId,
		UeCoreContext: &gnbUeContext.context,
		SourceGnbId:   oldGnb.controlPlaneInfo.gnbId,
	}, newGnb.controlPlaneInfo.gnbId, true)

	// send newGnb, conn to ue
	gnbUeContext.sendMsgToUe(&model.RLinkHandoverPrepareRequest{
		PrUeId:      gnbUeContext.prUeId,
		Conn:        conn,
		TargetGnbId: newGnb.controlPlaneInfo.gnbId,
	})

	newGnb.ueHoStatusPool.Store(prUeId, false)
}

func TriggerNgapHandover(sourceGnb *GnbContext, targetGnb *GnbContext, prUeId int64) {
	sourceGnb.Info("Initiating NGAP UE Handover")

	ue, err := sourceGnb.getGnbUeByPrUeId(prUeId)
	if err != nil {
		sourceGnb.Error("Error getting UE from PR UE ID: %s", err.Error())
		return
	}
	pduSessions := ue.context.PduSession
	PLMNIdentity := targetGnb.getPLMNIdentityInBytes()
	TAC := targetGnb.getTacInBytes()
	transfer := getSourceToTargetTransparentTransfer(sourceGnb, targetGnb, pduSessions, ue.prUeId)

	msg := &ies.HandoverRequired{
		AMFUENGAPID:  ue.amfUeNgapId,
		RANUENGAPID:  ue.ranUeNgapId,
		HandoverType: ies.HandoverType{Value: ies.HandoverTypeIntra5Gs},
		Cause: ies.Cause{
			Choice:       ies.CausePresentRadionetwork,
			RadioNetwork: &ies.CauseRadioNetwork{Value: ies.CauseRadioNetworkHandoverdesirableforradioreason},
		},
		PDUSessionResourceListHORqd: make([]ies.PDUSessionResourceItemHORqd, len(pduSessions)),
		TargetID: ies.TargetID{
			Choice: ies.TargetIDPresentTargetrannodeid,
			TargetRANNodeID: &ies.TargetRANNodeID{
				GlobalRANNodeID: ies.GlobalRANNodeID{
					Choice: ies.GlobalRANNodeIDPresentGlobalgnbId,
					GlobalGNBID: &ies.GlobalGNBID{
						PLMNIdentity: PLMNIdentity,
						GNBID: ies.GNBID{
							Choice: ies.GNBIDPresentGnbId,
							GNBID: &aper.BitString{
								Bytes:   targetGnb.getGnbIdInBytes(),
								NumBits: uint64(len(targetGnb.getGnbIdInBytes()) * 8),
							},
						},
					},
				},
				SelectedTAI: ies.TAI{
					PLMNIdentity: PLMNIdentity,
					TAC:          TAC,
				},
			},
		},
		SourceToTargetTransparentContainer: transfer,
	}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		//PDU SessionResource Admittedy Item
		PDUSessionID := pduSession.PduSessionId

		transfer := ies.HandoverRequiredTransfer{}
		var buf []byte
		var err error
		if buf, err = transfer.Encode(); err != nil {
			sourceGnb.Warn("err encode HandoverRequiredBuilder <- HandoverRequiredTransfer ")
		}

		msg.PDUSessionResourceListHORqd = append(msg.PDUSessionResourceListHORqd,
			ies.PDUSessionResourceItemHORqd{
				PDUSessionID:             PDUSessionID,
				HandoverRequiredTransfer: buf,
			})
	}

	if len(msg.PDUSessionResourceListHORqd) == 0 {
		sourceGnb.Error("No PDU Session to set up in InitialContextSetupResponse. NGAP Handover requires at least a PDU Session.")
	}

	ue.handoverTargetGnb = targetGnb
	ngapPdu, err := ngap.NgapEncode(msg)
	if err != nil {
		sourceGnb.Info("Error sending Handover Required: %s", err.Error())
	}

	sourceGnb.LogNgapSend("HandoverRequired")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		sourceGnb.Error("Error sending Handover Required: %s", err.Error())
	}
}
