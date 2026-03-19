package gnbcontext

import (
	"errors"
	"stormsim/internal/common/logger"
	"stormsim/internal/transport/rlink"
	"stormsim/internal/transport/sctpngap"
	"stormsim/pkg/model"
	"strconv"
	"sync"

	"github.com/lvdund/ngap/ies"
	"github.com/reogac/nas"
)

// UE main states in the GNB Context.
const (
	UE_INITIALIZED uint8 = iota
	UE_ONGOING
	UE_READY
	UE_DOWN
)

type GnbUeContext struct {
	*logger.Logger
	msin              string
	ranUeNgapId       int64              // Identifier for UE in GNB Context.
	amfUeNgapId       int64              // Identifier for UE in AMF Context.
	amfId             int64              // Identifier for AMF in UE/GNB Context.
	state             uint8              // State of UE in NAS/GNB Context.
	sctpConnection    *sctpngap.SctpConn // AMF Sctp association in using by the UE.
	rlinkConn         *rlink.Connection  // RLink connection between ue vs gnb
	prUeId            int64              // StormSim unique UE ID
	tmsi              *nas.Guti
	context           model.UeCoreContext
	lock              sync.Mutex
	handoverTargetGnb *GnbContext // Handover gnb
}

func (ue *GnbUeContext) createUeContext(plmn string, imeisv string, allowednssai []model.Snssai, ueSecCaps *ies.UESecurityCapabilities) {
	if plmn != "not informed" {
		ue.context.MobilityInfo.Mcc, ue.context.MobilityInfo.Mnc = convertMccMnc(plmn)
	} else {
		ue.context.MobilityInfo.Mcc = plmn
		ue.context.MobilityInfo.Mnc = plmn
	}

	ue.context.MaskedIMEISV = imeisv
	ue.context.AllowedSnssai = allowednssai
	ue.context.UeSecurityCapabilities = ueSecCaps
}

func (ue *GnbUeContext) CopyFromPreviousContext(amfUeNgapId *int64, ueCoreContext *model.UeCoreContext) {
	ue.amfUeNgapId = *amfUeNgapId
	ue.context = *ueCoreContext
}

func (ue *GnbUeContext) createPduSession(
	pduSessionId int64,
	upfIp string,
	sst string,
	sd string,
	pduType uint64,
	qosId int64,
	priArp int64,
	fiveQi int64,
	ulTeid uint32,
	dlTeid uint32,
) (*model.GnbPDUSessionContext, error) {

	if pduSessionId < 1 || pduSessionId >= 16 {
		return nil, errors.New("Invalid PDU Session Id [1,15]: " + strconv.FormatInt(pduSessionId, 10))
	}

	if ue.context.PduSession[pduSessionId] != nil {
		return nil, errors.New("Unable to create PDU Session " + strconv.FormatInt(pduSessionId, 10) + " as such PDU Session already exists")
	}

	var pduSession = new(model.GnbPDUSessionContext)
	pduSession.PduSessionId = pduSessionId
	pduSession.UpfIp = upfIp
	if !ue.isWantedNssai(sst, sd) {
		return nil, errors.New("Unable to create PDU Session, slice " + string(sst) + string(sd) + " is not selected for current UE")
	}
	pduSession.PduType = pduType
	pduSession.QosId = qosId
	pduSession.PriArp = priArp
	pduSession.FiveQi = fiveQi
	pduSession.UplinkTeid = ulTeid
	pduSession.DownlinkTeid = dlTeid
	pduSession.Sst = sst
	pduSession.Sd = sd

	ue.context.PduSession[pduSessionId] = pduSession

	return pduSession, nil
}

func (ue *GnbUeContext) getPduSession(pduSessionId int64) (*model.GnbPDUSessionContext, error) {
	if pduSessionId < 1 || pduSessionId >= 16 {
		return nil, errors.New("Invalid PDU Session Id [1,15]: " + strconv.FormatInt(pduSessionId, 10))
	}

	return ue.context.PduSession[pduSessionId], nil
}

func (ue *GnbUeContext) deletePduSession(pduSessionId int64) error {
	if pduSessionId < 1 || pduSessionId >= 16 {
		return errors.New("Invalid PDU Session Id [1,15]: " + strconv.FormatInt(pduSessionId, 10))
	}

	ue.context.PduSession[pduSessionId] = nil

	return nil
}

func (ue *GnbUeContext) getSelectedNssai(pduSessionId int64) (string, string) {
	pduSession := ue.context.PduSession[pduSessionId]
	if pduSession != nil {
		return pduSession.Sst, pduSession.Sd
	}

	return "NSSAI was not selected", "NSSAI was not selected"
}

func (ue *GnbUeContext) isWantedNssai(sst string, sd string) bool {
	if len(ue.context.AllowedSnssai) > 0 {
		for _, snssai := range ue.context.AllowedSnssai {
			if snssai.Sd == sd && snssai.Sst == sst {
				return true
			}
		}
	}
	return false
}
