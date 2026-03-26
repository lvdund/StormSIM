package gnbcontext

import (
	"encoding/binary"
	"stormsim/pkg/model"

	"github.com/lvdund/ngap/ies"
	"github.com/lvdund/ngap/utils"
)

func getPDUSessionResourceSetupResponseTransfer(ipv4 string, teid uint32, qosId int64) []byte {
	data := ies.PDUSessionResourceSetupResponseTransfer{}

	dowlinkTeid := make([]byte, 4)
	binary.BigEndian.PutUint32(dowlinkTeid, teid)
	ipNgap := utils.IPAddressToNgap(ipv4, "")
	data.DLQosFlowPerTNLInformation = ies.QosFlowPerTNLInformation{
		UPTransportLayerInformation: ies.UPTransportLayerInformation{
			Choice: ies.UPTransportLayerInformationPresentGtptunnel,
			GTPTunnel: &ies.GTPTunnel{
				GTPTEID:               dowlinkTeid,
				TransportLayerAddress: ipNgap,
			},
		},
		AssociatedQosFlowList: []ies.AssociatedQosFlowItem{{QosFlowIdentifier: qosId}},
	}

	var buf []byte
	var err error
	if buf, err = data.Encode(); err != nil {
		return nil
	}
	return buf
}

func (gnb *GnbContext) getHandoverRequestAcknowledgeTransfer(pduSession *model.GnbPDUSessionContext) []byte {
	data := ies.HandoverRequestAcknowledgeTransfer{}

	downlinkTeid := make([]byte, 4)
	binary.BigEndian.PutUint32(downlinkTeid, pduSession.DownlinkTeid)
	ip := utils.IPAddressToNgap(gnb.dataPlaneInfo.gnbIp, "")
	data.DLNGUUPTNLInformation = ies.UPTransportLayerInformation{
		Choice: ies.UPTransportLayerInformationPresentGtptunnel,
		GTPTunnel: &ies.GTPTunnel{
			GTPTEID:               downlinkTeid,
			TransportLayerAddress: ip,
		},
	}
	// FIX: why free5gc need this info
	data.DLForwardingUPTNLInformation = &ies.UPTransportLayerInformation{
		Choice: ies.UPTransportLayerInformationPresentGtptunnel,
		GTPTunnel: &ies.GTPTunnel{
			GTPTEID:               []byte{0, 0, 0, 1},
			TransportLayerAddress: ip,
		},
	}

	data.QosFlowSetupResponseList = []ies.QosFlowItemWithDataForwarding{
		{QosFlowIdentifier: 1},
	}

	var buf []byte
	var err error
	if buf, err = data.Encode(); err != nil {
		gnb.Fatal("aper MarshalWithParams error in GetHandoverRequestAcknowledgeTransfer: %+v", err)
	}
	return buf
}

func getTargetToSourceTransparentTransfer() []byte {
	data := ies.TargetNGRANNodeToSourceNGRANNodeTransparentContainer{
		RRCContainer: []byte("\x00\x00\x11"),
	}
	var buf []byte
	var err error
	if buf, err = data.Encode(); err != nil {
		return nil
	}
	return buf
}

func getSourceToTargetTransparentTransfer(
	sourceGnb *GnbContext,
	targetGnb *GnbContext,
	pduSessions [16]*model.GnbPDUSessionContext,
	prUeId int64,
) []byte {
	data := buildSourceToTargetTransparentTransfer(sourceGnb, targetGnb, pduSessions, prUeId)
	var buf []byte
	var err error
	if buf, err = data.Encode(); err != nil {
		sourceGnb.Fatal("aper MarshalWithParams error in GetSourceToTargetTransparentTransfer: %+v", err)
	}
	return buf
}

func buildSourceToTargetTransparentTransfer(
	sourceGnb *GnbContext,
	targetGnb *GnbContext,
	pduSessions [16]*model.GnbPDUSessionContext,
	prUeId int64,
) (data ies.SourceNGRANNodeToTargetNGRANNodeTransparentContainer) {
	data = ies.SourceNGRANNodeToTargetNGRANNodeTransparentContainer{}

	data.RRCContainer = []byte("\x00\x00\x11")
	data.IndexToRFSP = &prUeId

	data.PDUSessionResourceInformationList = []ies.PDUSessionResourceInformationItem{}
	for _, pduSession := range pduSessions {
		if pduSession == nil {
			continue
		}
		data.PDUSessionResourceInformationList = append(
			data.PDUSessionResourceInformationList,
			ies.PDUSessionResourceInformationItem{
				PDUSessionID:           pduSession.PduSessionId,
				QosFlowInformationList: []ies.QosFlowInformationItem{{QosFlowIdentifier: 1}},
			},
		)
	}
	if len(data.PDUSessionResourceInformationList) == 0 {
		sourceGnb.Error("[GNB][NGAP] No PDU Session to set up in InitialContextSetupResponse: NGAP Handover requires at least a PDU Session.")
		data.PDUSessionResourceInformationList = nil
	}

	PLMNIdentity := targetGnb.getPLMNIdentityInBytes()
	NRCellIdentity := targetGnb.getNRCellIdentity()
	data.TargetCellID = ies.NGRANCGI{
		Choice: ies.NGRANCGIPresentNrCgi,
		NRCGI: &ies.NRCGI{
			PLMNIdentity:   PLMNIdentity,
			NRCellIdentity: NRCellIdentity,
		},
	}

	PLMNIdentity = sourceGnb.getPLMNIdentityInBytes()
	NRCellIdentity = sourceGnb.getNRCellIdentity()
	data.UEHistoryInformation = []ies.LastVisitedCellItem{
		{
			LastVisitedCellInformation: ies.LastVisitedCellInformation{
				Choice: ies.LastVisitedCellInformationPresentNgrancell,
				NGRANCell: &ies.LastVisitedNGRANCellInformation{
					GlobalCellID: ies.NGRANCGI{
						Choice: ies.NGRANCGIPresentNrCgi,
						NRCGI: &ies.NRCGI{
							PLMNIdentity:   PLMNIdentity,
							NRCellIdentity: NRCellIdentity,
						},
					},
					CellType: ies.CellType{
						CellSize: ies.CellSize{Value: ies.CellSizeVerysmall},
					},
					TimeUEStayedInCell: 10,
				},
			},
		},
	}
	return data
}
