package gnbcontext

import (
	"encoding/binary"
	"stormsim/pkg/model"

	"github.com/lvdund/ngap"
	"github.com/lvdund/ngap/aper"
	"github.com/lvdund/ngap/ies"
)

func (gnb *GnbContext) buildInitialUeMessage(nasPdu []byte, ue *GnbUeContext) ([]byte, error) {
	gnb.LogNgapSend("InitialUEMessage")
	msg := ies.InitialUEMessage{
		RANUENGAPID: ue.ranUeNgapId,
		NASPDU:      nasPdu,
	}

	plmnid := gnb.getPLMNIdentityInBytes()
	cellid := gnb.getNRCellIdentity()
	plmnid_tai := gnb.getPlmnInOctets()
	tac := gnb.getTacInBytes()
	msg.UserLocationInformation = ies.UserLocationInformation{
		Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
		UserLocationInformationNR: &ies.UserLocationInformationNR{
			NRCGI: ies.NRCGI{
				PLMNIdentity:   plmnid,
				NRCellIdentity: cellid,
			},
			TAI: ies.TAI{
				PLMNIdentity: plmnid_tai,
				TAC:          tac,
			},
		},
	}

	msg.RRCEstablishmentCause = ies.RRCEstablishmentCause{Value: ies.RRCEstablishmentCauseMosignalling}
	guti5g := ue.tmsi
	// 5G-S-TSMI (optional)
	if guti5g != nil {
		var tmsiBytes [4]byte
		var amfSetBytes [2]byte

		binary.BigEndian.PutUint32(tmsiBytes[:], guti5g.Tmsi)
		binary.BigEndian.PutUint16(amfSetBytes[:], guti5g.AmfId.GetSet())

		msg.FiveGSTMSI = &ies.FiveGSTMSI{
			AMFSetID: aper.BitString{
				Bytes:   amfSetBytes[:],
				NumBits: 10,
			},
			AMFPointer: aper.BitString{
				Bytes:   []byte{guti5g.AmfId.GetPointer()},
				NumBits: 6,
			},
			FiveGTMSI: tmsiBytes[:],
		}
	}

	// UE Context Request (optional)
	msg.UEContextRequest = &ies.UEContextRequest{Value: ies.UEContextRequestRequested}

	return ngap.NgapEncode(&msg)
}

func (gnb *GnbContext) buildUplinkNasTransport(nasPdu []byte, ue *GnbUeContext) ([]byte, error) {
	gnb.LogNgapSend("UplinkNASTransport")
	msg := ies.UplinkNASTransport{}

	// AMF UE NGAP ID
	msg.AMFUENGAPID = ue.amfUeNgapId

	// RAN UE NGAP ID
	msg.RANUENGAPID = ue.ranUeNgapId

	// NAS-PDU
	msg.NASPDU = nasPdu

	// User Location Information
	plmnid := gnb.getPLMNIdentityInBytes()
	cellid := gnb.getNRCellIdentity()
	tac := gnb.getTacInBytes()
	msg.UserLocationInformation = ies.UserLocationInformation{
		Choice: ies.UserLocationInformationPresentUserlocationinformationnr,
		UserLocationInformationNR: &ies.UserLocationInformationNR{
			NRCGI: ies.NRCGI{
				PLMNIdentity:   plmnid,
				NRCellIdentity: cellid,
			},
			TAI: ies.TAI{
				PLMNIdentity: plmnid,
				TAC:          tac,
			},
		},
	}
	return ngap.NgapEncode(&msg)
}

func (gnb *GnbContext) sendInitialContextSetupResponse(ue *GnbUeContext) {
	gnb.Info("Initiating Initial Context Setup Response")

	pduSessions := ue.context.PduSession
	msg := &ies.InitialContextSetupResponse{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
	}
	sessions := []ies.PDUSessionResourceSetupItemCxtRes{}
	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		pdusessionid := pduSession.PduSessionId
		transfer := getPDUSessionResourceSetupResponseTransfer(gnb.dataPlaneInfo.gnbIp, pduSession.DownlinkTeid, pduSession.QosId)

		sessions = append(sessions,
			ies.PDUSessionResourceSetupItemCxtRes{
				PDUSessionID:                            pdusessionid,
				PDUSessionResourceSetupResponseTransfer: transfer,
			})
	}

	if len(sessions) == 0 {
		gnb.Info("No PDU Session to set up in InitialContextSetupResponse.")
	} else {
		msg.PDUSessionResourceSetupListCxtRes = sessions
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Initial Context Setup Response: ", err)
		return
	}

	gnb.LogNgapSend("InitialContextSetupResponse")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Initial Context Setup Response: ", err)
	}
}

func (gnb *GnbContext) sendPduSessionResourceSetupResponse(pduSessions []*model.GnbPDUSessionContext, ue *GnbUeContext) {
	gnb.Info("Initiating PDU Session Resource Setup Response")
	msg := &ies.PDUSessionResourceSetupResponse{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
	}
	sessions := []ies.PDUSessionResourceSetupItemSURes{}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		plmnid := pduSession.PduSessionId
		transfer := getPDUSessionResourceSetupResponseTransfer(gnb.dataPlaneInfo.gnbIp, pduSession.DownlinkTeid, pduSession.QosId)

		sessions = append(sessions,
			ies.PDUSessionResourceSetupItemSURes{
				PDUSessionID:                            plmnid,
				PDUSessionResourceSetupResponseTransfer: transfer,
			})
	}
	if len(sessions) > 0 {
		msg.PDUSessionResourceSetupListSURes = sessions
	}
	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create PDU Session Resource Setup Response: ", err)
		return
	}

	ue.state = UE_READY

	gnb.LogNgapSend("PDUSessionResourceSetupResponse")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending PDU Session Resource Setup Response.: ", err)
	}
}

func (gnb *GnbContext) sendPduSessionReleaseResponse(pduSessionIds []int64, ue *GnbUeContext) {
	gnb.Info("Initiating PDU Session Release Response: ", ue.amfId, ue.ranUeNgapId)

	if len(pduSessionIds) == 0 {
		gnb.Fatal("Trying to send a PDU Session Release Reponse for no PDU Session")
	}
	msg := ies.PDUSessionResourceReleaseResponse{
		AMFUENGAPID: ue.amfId,
		RANUENGAPID: ue.ranUeNgapId,
	}

	msg.PDUSessionResourceReleasedListRelRes = []ies.PDUSessionResourceReleasedItemRelRes{}
	for _, pduSessionId := range pduSessionIds {
		msg.PDUSessionResourceReleasedListRelRes = append(msg.PDUSessionResourceReleasedListRelRes, ies.PDUSessionResourceReleasedItemRelRes{
			PDUSessionID: pduSessionId,
			PDUSessionResourceReleaseResponseTransfer: []byte{00},
		})
	}

	if len(msg.PDUSessionResourceReleasedListRelRes) == 0 {
		gnb.Info("PDUSessionResourceReleasedListRelRes empty")
		msg.PDUSessionResourceReleasedListRelRes = nil
	}

	ngapPdu, err := ngap.NgapEncode(&msg)
	if err != nil {
		gnb.Error("Error creating PDU Session Release Response: ", err)
		return
	}

	gnb.LogNgapSend("PDUSessionResourceReleaseResponse")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending PDU Session Release Response: ", err)
	}
}

func (gnb *GnbContext) sendUeContextReleaseComplete(ue *GnbUeContext) {
	gnb.Info("Initiating UE Context Release Complete")
	msg := ies.UEContextReleaseComplete{
		AMFUENGAPID: ue.amfUeNgapId,
		RANUENGAPID: ue.ranUeNgapId,
	}
	ngapPdu, err := ngap.NgapEncode(&msg)

	if err != nil {
		gnb.Error("Error create UE Context Release Complete: ", err)
		return
	}

	gnb.LogNgapSend("UEContextReleaseComplete")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending UE Context Complete: ", err)
	}

	gnb.deleteGnBUe(ue)
}
func (gnb *GnbContext) sendHandoverRequestAcknowledge(ue *GnbUeContext) {
	gnb.Info("Initiating Handover Request Acknowledge")
	pduSessions := ue.context.PduSession
	targetToSourceTransparentContainer := getTargetToSourceTransparentTransfer()

	msg := &ies.HandoverRequestAcknowledge{
		AMFUENGAPID:                        ue.amfUeNgapId,
		RANUENGAPID:                        ue.ranUeNgapId,
		PDUSessionResourceAdmittedList:     make([]ies.PDUSessionResourceAdmittedItem, len(pduSessions)),
		TargetToSourceTransparentContainer: targetToSourceTransparentContainer,
	}

	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		//PDU SessionResource Admittedy Item
		PDUSessionID := pduSession.PduSessionId
		HandoverRequestAcknowledgeTransfer := gnb.getHandoverRequestAcknowledgeTransfer(pduSession)
		msg.PDUSessionResourceAdmittedList = append(msg.PDUSessionResourceAdmittedList, ies.PDUSessionResourceAdmittedItem{
			PDUSessionID:                       PDUSessionID,
			HandoverRequestAcknowledgeTransfer: HandoverRequestAcknowledgeTransfer,
		})
	}

	if len(msg.PDUSessionResourceAdmittedList) == 0 {
		gnb.Info("No admitted PDU Session")
	}

	ngapPdu, err := ngap.NgapEncode(msg)

	if err != nil {
		gnb.Error("Error create Handover Request Acknowledge: ", err)
		return
	}

	gnb.LogNgapSend("HandoverRequestAcknowledge")
	err = ue.sendNgap(ngapPdu)
	if err != nil {
		gnb.Error("Error sending Handover Request Acknowledge: ", err)
		return
	}

	if _, ok := gnb.ueHoStatusPool.Load(ue.prUeId); ok {
		gnb.ueHoStatusPool.Store(ue.prUeId, true)
	}
}
