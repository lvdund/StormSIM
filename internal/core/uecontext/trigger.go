package uecontext

import (
	"fmt"
	"stormsim/internal/common/fsm"
	"stormsim/internal/core/gnbcontext"
	"stormsim/pkg/model"

	"github.com/reogac/nas"
)

// func SendRemoteEvent(msg *EventUeData) {
// 	ue := msg.Ue
//
// 	switch msg.EventType {
// 	case RegisterInit:
// 		ue.SendEventMm(fsm.NewEmptyEventData(RegisterInit))
// 	// case InitRegistrationRequestEvent:
// 	// 	ue.SendEventMm(fsm.NewEmptyEventData(InitRegistrationRequestEvent))
// 	case DeregistraterInit:
// 		ue.SendEventMm(fsm.NewEventData(DeregistraterInit, &msg.Params))
// 	case PduSessionInit:
// 		ue.SendEventMm(fsm.NewEmptyEventData(PduSessionInit))
// 	case DestroyPduSession:
// 		ue.SendEventMm(fsm.NewEmptyEventData(DestroyPduSession))
// 	case IdleInit:
// 		ue.SendEventMm(fsm.NewEmptyEventData(IdleInit))
// 	case ServiceRequestInit:
// 		ue.SendEventMm(fsm.NewEmptyEventData(ServiceRequestInit))
// 	case Terminate:
// 		if len(ue.expFile) > 0 {
// 			if expFile, err := os.OpenFile(
// 				ue.expFile,
// 				os.O_WRONLY|os.O_CREATE|os.O_APPEND,
// 				0644,
// 			); err != nil {
// 				ue.Error("Failed to create logfile %s", ue.expFile)
// 			} else {
// 				logExpResults(expFile, ue)
// 				expFile.Close()
// 			}
// 		}
// 		ue.SendEventMm(fsm.NewEventData(Terminate, &msg.Params))
// 	case Kill:
// 		ue.Terminate()
// 	default:
// 		ue.Info("UE not support this remote event!")
// 	}
// }

func (ue *UeContext) triggerInitRegistration() (err error) {
	ue.Info("Initiating Registration")

	msg := &nas.RegistrationRequest{
		UeSecurityCapability: ue.secCap,
	}
	msg.RegistrationType = nas.NewRegistrationType(true, nas.RegistrationType5GSInitialRegistration)

	if ue.guti != nil {
		msg.MobileIdentity = nas.MobileIdentity{
			Id: ue.guti,
		}
	} else {
		msg.MobileIdentity = ue.suci
	}
	if ue.secCtx != nil {
		msg.Ngksi = *ue.secCtx.NgKsi()
	} else {
		msg.Ngksi.Id = 7
	}

	//FIX: open5gs cannot read this field
	var gmmCap [13]byte ////////////////////////
	gmmCap[0] = 0x07
	msg.GmmCapability = new(nas.GmmCapability)
	msg.GmmCapability.Bytes = gmmCap[:]

	var pduFlag [16]bool
	hasPdu := false
	for i, pduSession := range ue.sessions {
		if pduSession != nil {
			hasPdu = true
			pduFlag[i] = true
		}
	}

	if hasPdu {
		msg.UplinkDataStatus = new(nas.UplinkDataStatus)
		msg.UplinkDataStatus.Set(pduFlag)

		msg.PduSessionStatus = new(nas.PduSessionStatus)
		msg.PduSessionStatus.Set(pduFlag)
	}

	msg.SetSecurityHeader(nas.NasSecNone)

	//FIX: open5gs cannot read this field
	msg.RequestedNssai = &nas.Nssai{ ///////////////////////
		List: []nas.SNssai{{
			Sst: ue.snssai.NasType().Sst,
			Sd:  ue.snssai.NasType().Sd,
		}},
	}

	nasPdu, _ := nas.EncodeMm(nil, msg)
	ue.nasPdu = make([]byte, len(nasPdu)) //Keep a copy of this registration request
	copy(ue.nasPdu, nasPdu)

	if hasPdu {
		//encrypt the request
		nasCtx := ue.getNasContext()                   //must be non-nil
		cipher, _ := nasCtx.EncryptMmContainer(nasPdu) //ignore error for now
		//embed the encrypted request into the original one
		msg.NasMessageContainer = cipher
		//reset UplinkDataStatus and PduSessionStatus
		msg.UplinkDataStatus = nil
		msg.PduSessionStatus = nil
		//now plaintext-encode again
		nasPdu, err = nas.EncodeMm(nil, msg)
		if err != nil {
			return
		}
	}
	// send to GNB.
	ue.LogSend("nas", "RegistrationRequest")
	ue.sendNas(nasPdu)

	return nil
}

func (ue *UeContext) triggerInitDeregistration(deregisterType int) (err error) {
	ue.Info("Initiating Deregistration")
	var nasPdu []byte

	if ue.secCtx == nil {
		return fmt.Errorf("Missing security context")
	}

	msg := &nas.DeregistrationRequestFromUe{
		Ngksi: *ue.secCtx.NgKsi(),
	}

	switch deregisterType {
	case 0: // not switch off
		msg.DeRegistrationType.SetSwitchOff(false)
	case 1: // switch off
		msg.DeRegistrationType.SetSwitchOff(true)
	}

	msg.DeRegistrationType.SetReregistration(false)
	msg.DeRegistrationType.SetAccessType(nas.AccessType3GPP)
	if ue.guti != nil {
		msg.MobileIdentity.Id = ue.guti
	} else {
		msg.MobileIdentity = ue.suci
	}

	nasCtx := ue.getNasContext() //must be non nil
	msg.SetSecurityHeader(nas.NasSecBoth)
	if nasPdu, err = nas.EncodeMm(nasCtx, msg); err != nil {
		return err
	} else {
		// send to GNB.
		ue.LogSend("nas", "DeregistrationRequestFromUE")
		ue.sendNas(nasPdu)
	}
	if deregisterType == 1 {
		ue.Terminate()
	}
	return nil
}

func (ue *UeContext) triggerInitPduSessionRequest(params *map[string]any) {
	ue.Info("Initiating New PDU Session")
	pduSession, err := ue.createPDUSession()
	if err != nil {
		ue.Fatal("[UE][NAS] %v", err)
		return
	}

	pduSession.Info("PDU Session Initiating")
	pduSession.SendEventSm(fsm.NewEventData(model.InitPduSessionEstablishmentRequestEvent, params))
}

func (ue *UeContext) triggerInitPduSessionRequestInner(
	pduSession *PduSession,
	params *map[string]any,
) {
	n1Sm := new(nas.PduSessionEstablishmentRequest)
	pduType := nas.PduSessionTypeIpv4
	n1Sm.PduSessionType = &pduType
	n1Sm.IntegrityProtectionMaximumDataRate = nas.NewIntegrityProtectionMaximumDataRate(0xff, 0xff)
	/*
		//TODO: @Dung please implement this : add Extended PCOs: refer to SMF (handle
			//PduSessionEstablishmentRequest)
					msg.ExtendedProtocolConfigurationOptions = nasType.NewExtendedProtocolConfigurationOptions(nasMessage.PDUSessionEstablishmentRequestExtendedProtocolConfigurationOptionsType)
					protocolConfigurationOptions := nasConvert.NewProtocolConfigurationOptions()
					protocolConfigurationOptions.AddIPAddressAllocationViaNASSignallingUL()
					protocolConfigurationOptions.AddDNSServerIPv4AddressRequest()
					protocolConfigurationOptions.AddDNSServerIPv6AddressRequest()
					pcoContents := protocolConfigurationOptions.Marshal()
					pcoContentsLength := len(pcoContents)
					msg.ExtendedProtocolConfigurationOptions.SetLen(uint16(pcoContentsLength))
					msg.ExtendedProtocolConfigurationOptions.SetExtendedProtocolConfigurationOptionsContents(pcoContents)
	*/
	// n1Sm.ExtendedProtocolConfigurationOptions = &nas.ExtendedProtocolConfigurationOptions{}
	// pcoContents := []byte{128, 0, 10, 0, 0, 13, 0, 0, 3, 0}
	// n1Sm.ExtendedProtocolConfigurationOptions.AddUnit(nas.PcoUnit{
	// 	Id: 10,
	// 	Content: pcoContents,
	// })

	n1Sm.SetPti(1)
	n1Sm.SetSessionId(pduSession.id)

	n1SmPdu, _ := nas.EncodeSm(n1Sm)
	requestType := nas.UlNasTransportRequestTypeInitialRequest
	ue.LogSend("nas", "PduSessionEstablishmentRequest")
	ue.sendN1Sm(n1SmPdu, pduSession.id, &requestType, params)
}

func (ue *UeContext) triggerInitPduSessionReleaseRequest(pduSession *PduSession) {
	ue.Info("Initiating Release of PDU Session %d", pduSession.id)

	if pduSession.state_sm.CurrentState() != model.PDUSessionActive {
		ue.Warn("Skipping releasing the PDU Session ID %d as it's not active: %v", pduSession.id, pduSession.state_sm.CurrentState())
		return
	}
	n1Sm := new(nas.PduSessionReleaseRequest)

	n1Sm.SetPti(1)
	n1Sm.SetSessionId(pduSession.id)
	n1SmPdu, _ := nas.EncodeSm(n1Sm)
	ue.LogSend("nas", "PduSessionReleaseRequest")
	ue.sendN1Sm(n1SmPdu, pduSession.id, nil, nil)
}

func (ue *UeContext) triggerInitPduSessionReleaseComplete(pduSession *PduSession) {
	ue.Info("Initiating PDU Session Release Complete for PDU Session: %d", pduSession.id)

	if pduSession.state_sm.CurrentState() == model.PDUSessionInactive {
		ue.Warn("Unable to send PDU Session Release Complete for a PDU Session which is not inactive")
		return
	}
	n1Sm := new(nas.PduSessionReleaseComplete)

	n1Sm.SetPti(1) //must be same as received command message
	n1Sm.SetSessionId(pduSession.id)
	n1SmPdu, _ := nas.EncodeSm(n1Sm)
	ue.LogSend("nas", "PduSessionReleaseComplete")
	ue.sendN1Sm(n1SmPdu, pduSession.id, nil, nil)
}

func (ue *UeContext) triggerSwitchToIdle() {
	ue.Info("Switching to 5GMM-IDLE")
	gnbcontext.SendToGnb(ue.msin, &model.RlinkUeIdleInform{PrUeId: int64(ue.id)}, ue.gnbId, false)
}

func (ue *UeContext) triggerInitServiceRequest() {
	ue.Info("Initiating Service Request")

	msg := &nas.ServiceRequest{
		Ngksi:       ue.auth.ngKsi,
		STmsi:       ue.get5GTmsi(),
		ServiceType: nas.ServiceTypeData,
	}

	msg.SetSecurityHeader(nas.NasSecNone)
	//there must be pdu sessions

	var pduFlag [16]bool
	hasPdu := false
	for i, pduSession := range ue.sessions {
		if pduSession != nil {
			hasPdu = true
			pduFlag[i] = true
		}
	}
	if hasPdu {
		msg.UplinkDataStatus = new(nas.UplinkDataStatus)
		msg.UplinkDataStatus.Set(pduFlag)

		msg.PduSessionStatus = new(nas.PduSessionStatus)
		msg.PduSessionStatus.Set(pduFlag)
	}
	nasPdu, _ := nas.EncodeMm(nil, msg)
	//Keep a copy of this service request
	ue.nasPdu = make([]byte, len(nasPdu))
	copy(ue.nasPdu, nasPdu)

	if hasPdu {
		//encrypt the request
		nasCtx := ue.getNasContext()                   //must be non-nil
		cipher, _ := nasCtx.EncryptMmContainer(nasPdu) //ignore error for now
		//embed the encrypted request into the original one
		msg.NasMessageContainer = cipher
		//reset UplinkDataStatus and PduSessionStatus
		msg.UplinkDataStatus = nil
		msg.PduSessionStatus = nil
		//now plaintext-encode again
		nasPdu, _ = nas.EncodeMm(nil, msg)
	}

	// send to GNB.
	ue.LogSend("nas", "ServiceRequest")
	ue.sendNas(nasPdu)
}
