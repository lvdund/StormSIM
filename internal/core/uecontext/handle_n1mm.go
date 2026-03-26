package uecontext

import (
	"stormsim/internal/common/fsm"
	"stormsim/internal/core/uecontext/sec"
	"stormsim/pkg/model"

	"github.com/reogac/nas"
)

func (ue *UeContext) handleNasMsg(nasBytes []byte) {
	if len(nasBytes) == 0 {
		ue.Error("NAS message is empty")
		return
	}

	var nasMsg nas.NasMessage
	var err error
	if nasMsg, err = nas.Decode(ue.getNasContext(), nasBytes); err != nil {
		ue.Error("Decode Nas message failed: %s", err.Error())
		return
	}

	// var taskLog *logger.TaskStat = &logger.TaskStat{}
	//logger.TaskStats[ue.msin] = append(logger.TaskStats[ue.msin], taskLog)

	ue.handleNas_n1mm(&nasMsg)
}

func (ue *UeContext) handleNas_n1mm(nasMsg *nas.NasMessage) {
	gmm := nasMsg.Gmm
	if gmm == nil {
		ue.Error("NAS message is has no N1MM content")
		return
	}

	if ue.enableFuzz {
		Capture.CaptureMsgToUe(int(gmm.MsgType))
	}

	switch gmm.MsgType {
	case nas.AuthenticationRequestMsgType:
		ue.LogReceive("nas", "AuthenticationRequest")
		//taskLog.Task = "Task MM: handle Receive Authentication Request msg"
		ue.handleAuthenticationRequest(gmm.AuthenticationRequest)

	case nas.AuthenticationRejectMsgType:
		ue.LogReceive("nas", "AuthenticationReject")
		//taskLog.Task = "Task MM: handle Receive Authentication Reject msg"
		ue.handleAuthenticationReject(gmm.AuthenticationReject)

	case nas.IdentityRequestMsgType:
		ue.LogReceive("nas", "IdentityRequest")
		//taskLog.Task = "Task MM: handle Receive Identify Request msg"
		ue.handleIdentityRequest(gmm.IdentityRequest)

	case nas.SecurityModeCommandMsgType:
		ue.LogReceive("nas", "SecurityModeCommand")
		//taskLog.Task = "Task MM: handle Receive Security Mode Command msg"
		ue.handleSecurityModeCommand(gmm.SecurityModeCommand)

	case nas.RegistrationAcceptMsgType:
		ue.LogReceive("nas", "RegistrationAccept")
		//taskLog.Task = "Task MM: handle Receive Registration Accept msg"
		ue.handleRegistrationAccept(gmm.RegistrationAccept)
		ue.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.RegistrationAcceptEvent))

	case nas.ConfigurationUpdateCommandMsgType:
		ue.LogReceive("nas", "ConfigurationUpdateCommand")
		//taskLog.Task = "Task MM: handle Receive Configuration Update Command msg"
		ue.handleConfigurationUpdateCommand(gmm.ConfigurationUpdateCommand)

	case nas.DlNasTransportMsgType:
		ue.LogReceive("nas", "DlNasTransport")
		//taskLog.Task = "Task MM: handle Receive DL NAS Transport msg"
		ue.handleCause5GMM(gmm.DlNasTransport.GmmCause)
		ue.handleDlNasTransport(gmm.DlNasTransport)

	case nas.ServiceAcceptMsgType:
		ue.LogReceive("nas", "ServiceAccept")
		//taskLog.Task = "Task MM: handle Receive Service Accept msg"
		ue.handleServiceAccept(gmm.ServiceAccept)

	case nas.ServiceRejectMsgType:
		ue.LogReceive("nas", "ServiceReject")
		//taskLog.Task = "Task MM: handle Receive Service Reject msg"
		ue.handleCause5GMM(&gmm.ServiceReject.GmmCause)

	case nas.RegistrationRejectMsgType:
		ue.LogReceive("nas", "RegistrationReject")
		//taskLog.Task = "Task MM: handle Receive Registration Reject msg"
		ue.handleCause5GMM(&gmm.RegistrationReject.GmmCause)
		ue.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.RegistrationRejectEvent))

	case nas.GmmStatusMsgType:
		ue.LogReceive("nas", "5GMMStatus")
		//taskLog.Task = "Task MM: handle Receive Status 5GMM msg"
		ue.handleCause5GMM(&gmm.GmmStatus.GmmCause)

	case nas.DeregistrationAcceptFromUeMsgType:
		ue.LogReceive("nas", "DeregistrationAcceptFromUE")
		//taskLog.Task = "Task MM: handle Receive Deregister Accept msg"
		ue.handleDeregistrationAccept(gmm.DeregistrationAcceptFromUe)

	case nas.DeregistrationRequestToUeMsgType:
		ue.LogReceive("nas", "DeregistrationRequestToUE")
		//taskLog.Task = "Task MM: handle Receive Deregister Request from AMF msg"
		ue.handleDeregistrationRequestFromNetwork(gmm.DeregistrationRequestToUe)

	default:
		ue.Warn("Received unknown NAS message 0x%x", nasMsg.Gmm.MsgType)
		//taskLog.Task = fmt.Sprintf("Task MM: Received unknown NAS message 0x%x", nasMsg.Gmm.MsgType)
	}

	// endTime := time.Now()
	// duration := endTime.Sub(taskLog.Start)
	// taskLog.End = &endTime
	// taskLog.Durarion = &duration
}

func (ue *UeContext) handleCause5GMM(cause *uint8) {
	if cause != nil {
		ue.Error("UE received a 5GMM Failure, cause: %s", cause5GMMToString(uint8(*cause)))
	}
}

func (ue *UeContext) handleAuthenticationReject(message *nas.AuthenticationReject) {
	_ = message
	ue.Info("Authentication of UE failed")
	ue.sendEventMm(fsm.NewEmptyEventData(model.AuthFailEvent))
}

func (ue *UeContext) handleAuthenticationRequest(message *nas.AuthenticationRequest) {
	var responsePdu []byte
	var response nas.GmmMessage

	if message.Ngksi.Id == 7 {
		ue.Fatal("Error in Authentication Request, ngKSI not the expected value")
	}

	if len(message.Abba) == 0 {
		ue.Fatal("Error in Authentication Request, ABBA Content is empty")
	}
	if message.AuthenticationParameterRand == nil {
		ue.Fatal("Error in Authentication Request, RAND is missing")
	}

	if message.AuthenticationParameterAutn == nil {
		ue.Fatal("Error in Authentication Request, AUTN is missing")
	}
	// getting NgKsi, RAND and AUTN from the message.
	ue.auth.ngKsi = message.Ngksi
	ue.auth.rand = message.AuthenticationParameterRand
	ue.auth.milenage.SetRand(ue.auth.rand)

	autn := message.AuthenticationParameterAutn
	abba := message.Abba

	// getting resStar
	errCode, paramDat := ue.auth.processAuthenticationInfo(autn, abba)
	switch errCode {

	case AUTH_MAC_FAILURE:
		ue.Error("Authentication request validation: Failed: MAC mismatch")
		msg := &nas.AuthenticationFailure{
			GmmCause: nas.Cause5GMMMACFailure,
		}
		msg.SetSecurityHeader(nas.NasSecNone)
		response = msg
		ue.LogSend("nas", "AuthenticationFailure")
	case AUTH_SYNC_FAILURE:
		ue.Info("Authentication request validation: OK")
		ue.Error("SQN of the authentication request message: INVALID")
		ue.Error("Send authentication failure with Synch failure")
		msg := &nas.AuthenticationFailure{
			GmmCause:                       nas.Cause5GMMSynchFailure,
			AuthenticationFailureParameter: paramDat,
		}
		msg.SetSecurityHeader(nas.NasSecNone)
		response = msg
		ue.LogSend("nas", "AuthenticationFailure")

	case AUTH_SUCCESS:
		ue.Info("Authentication request validation: OK")
		ue.Info("SQN of the authentication request message: VALID")
		ue.Info("Send authentication response")
		msg := &nas.AuthenticationResponse{
			AuthenticationResponseParameter: paramDat,
		}
		msg.SetSecurityHeader(nas.NasSecNone)
		response = msg
		// create an inactive security context
		ue.secCtx = sec.NewSecurityContext(&ue.auth.ngKsi, ue.auth.kamf, false)
		ue.LogSend("nas", "AuthenticationResponse")
	}

	responsePdu, _ = nas.EncodeMm(nil, response)
	ue.sendNas(responsePdu)
}

func (ue *UeContext) handleSecurityModeCommand(message *nas.SecurityModeCommand) {
	//check for existing NgKsi
	if message.Ngksi.Id == 7 || ue.auth.ngKsi.Id != message.Ngksi.Id || ue.auth.ngKsi.Tsc != message.Ngksi.Tsc {
		ue.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.SecurityModeFailEvent))
		ue.Error("Error in Security Mode Command, ngKSI not the expected value")
		return
	}

	algs := message.SelectedNasSecurityAlgorithms
	switch algs.EncAlg() {
	case nas.AlgCiphering128NEA0:
		ue.Info("Selected ciphering algorithm: 5G-0")
	case nas.AlgCiphering128NEA1:
		ue.Info("Selected ciphering algorithm: 128-5G-1")
	case nas.AlgCiphering128NEA2:
		ue.Info("Selected ciphering algorithm: 128-5G-2")
	case nas.AlgCiphering128NEA3:
		ue.Info("Selected ciphering algorithm: 128-5G-3")
	}
	switch algs.IntAlg() {
	case nas.AlgIntegrity128NIA0:
		ue.Info("Selected integrity algorithm: 5G-IA0")
	case nas.AlgIntegrity128NIA1:
		ue.Info("Selected integrity algorithm: 128-5G-IA1")
	case nas.AlgIntegrity128NIA2:
		ue.Info("Selected integrity algorithm: 128-5G-IA2")
	case nas.AlgIntegrity128NIA3:
		ue.Info("Selected integrity algorithm: 128-5G-IA3")
	}

	rinmr := false
	if message.AdditionalSecurityInformation != nil {
		// checking BIT RINMR that triggered registration request in security mode complete.
		rinmr = message.AdditionalSecurityInformation.GetRetransmission()
		ue.Info("Have Additional Secutity Information, retransmission = %v", rinmr)
	}

	//derive NasContext keys (then the security context is activated)
	ue.secCtx.NasContext(true).DeriveKeys(algs.EncAlg(), algs.IntAlg(), ue.secCtx.Kamf())

	//TODO: Implement imeisv
	imeisv := nas.Imei{IsSv: true}
	imeisv.Parse("1110000000000000") //dummy imei
	response := &nas.SecurityModeComplete{
		Imeisv: &nas.MobileIdentity{
			Id: &imeisv,
		},
	}
	nasCtx := ue.getNasContext()
	rinmr = true //just for etrib5gc
	if rinmr {
		response.NasMessageContainer = ue.nasPdu
	}

	response.SetSecurityHeader(nas.NasSecBothNew)
	responsePdu, _ := nas.EncodeMm(nasCtx, response)
	ue.LogSend("nas", "SecurityModeComplete")
	ue.sendNas(responsePdu)
}

func (ue *UeContext) handleRegistrationAccept(message *nas.RegistrationAccept) {

	// change the state of ue for registered
	ue.Info("Handle Registration Accept")

	// saved 5g GUTI and others information.
	if message.Guti != nil {
		ue.set5gGuti(message.Guti)
	} else {
		ue.Warn("UE was not assigned a 5G-GUTI by AMF")
	}

	// use the slice allowed by the network in PDU session request
	if ue.snssai.Sst == 0 && message.AllowedNssai != nil {
		// check the allowed NSSAI received from the 5GC
		snssai := message.AllowedNssai.List[0] //very sloppy, need checking

		// update UE slice selected for PDU Session
		ue.snssai.Sst = int(snssai.Sst)
		ue.snssai.Sd = snssai.GetSd()

		ue.Warn("ALLOWED NSSAI: SST:%d SD:%s", ue.snssai.Sst, ue.snssai.Sd)
	}

	ue.Info("UE 5G GUTI: %s", ue.guti.String())

	// getting NAS registration complete.
	response := &nas.RegistrationComplete{}
	//TODO: set SORTransparentContainer if needed

	response.SetSecurityHeader(nas.NasSecBoth)
	nasCtx := ue.getNasContext() //must be non-nil
	responsePdu, _ := nas.EncodeMm(nasCtx, response)
	ue.LogSend("nas", "RegistrationComplete")
	ue.sendNas(responsePdu)
}

func (ue *UeContext) handleServiceAccept(message *nas.ServiceAccept) {
	// change the state of ue for registered
	// ue.setState(MM5G_REGISTERED)
}

func (ue *UeContext) handleDlNasTransport(message *nas.DlNasTransport) {

	if uint8(message.PayloadContainerType) != nas.PayloadContainerTypeN1SMInfo {
		ue.Fatal("Error in DL NAS Transport, Payload Container Type not expected value")
	}

	if message.PduSessionId == nil {
		ue.Fatal("Error in DL NAS Transport, PDU Session ID is missing")
	}

	//decode N1Sm message
	nasMsg, err := nas.Decode(nil, message.PayloadContainer)

	if err != nil {
		ue.Fatal("Error in DL NAS Transport, fail to decode N1Sm")
	}

	// var taskLog *logger.TaskStat = &logger.TaskStat{}
	//logger.TaskStats[ue.msin] = append(logger.TaskStats[ue.msin], taskLog)

	ue.handleNas_n1sm(&nasMsg)
}

func (ue *UeContext) handleIdentityRequest(message *nas.IdentityRequest) {

	switch uint8(message.IdentityType) {
	case nas.MobileIdentity5GSTypeSuci:
		ue.Info("Requested SUCI 5GS type")
	default:
		ue.Fatal("Only SUCI identity is supported for now inside StormSim")
	}

	rsp := &nas.IdentityResponse{
		MobileIdentity: ue.suci, //TODO: can be SUCI/IMEISV etc
	}
	nasCtx := ue.getNasContext()
	if nasCtx != nil {
		rsp.SetSecurityHeader(nas.NasSecBoth)
	} else {
		rsp.SetSecurityHeader(nas.NasSecNone)
	}

	if nasPdu, err := nas.EncodeMm(nasCtx, rsp); err != nil {
		ue.Fatal("Error encoding identity request: %v", err)
	} else {
		ue.LogSend("nas", "IdentityResponse")
		ue.sendNas(nasPdu)
	}
}

func (ue *UeContext) handleConfigurationUpdateCommand(message *nas.ConfigurationUpdateCommand) {
	_ = message
	ue.Info("Initiating Configuration Update Complete")
	msg := &nas.ConfigurationUpdateComplete{}
	nasCtx := ue.getNasContext()
	msg.SetSecurityHeader(nas.NasSecBoth)

	if nasPdu, err := nas.EncodeMm(nasCtx, msg); err != nil {
		ue.Fatal("Error encoding Configuration Update Complete: %v", err)
	} else {
		ue.LogSend("nas", "ConfigurationUpdateComplete")
		ue.sendNas(nasPdu)
	}
}

func (ue *UeContext) handleDeregistrationAccept(message *nas.DeregistrationAcceptFromUe) {
	_ = message
	ue.Info("delete security context after receiving deregistration accept")
	ue.resetSecurityContext()
}

func (ue *UeContext) handleDeregistrationRequestFromNetwork(message *nas.DeregistrationRequestToUe) {
	_ = message
	msg := &nas.DeregistrationAcceptToUe{}
	msg.SetSecurityHeader(nas.NasSecNone)
	if nasPdu, err := nas.EncodeMm(nil, msg); err != nil {
		ue.Fatal("Error encoding Deregistration Accept To Ue: %v", err)
	} else {
		ue.LogSend("nas", "DeregistrationAcceptToUE")
		ue.sendNas(nasPdu)
	}
}
