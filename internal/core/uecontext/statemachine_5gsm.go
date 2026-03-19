package uecontext

import (
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/stats"
	"stormsim/pkg/model"

	"github.com/alitto/pond/v2"
)

func initPduFSM(w pond.Pool) *fsm.Fsm {
	transitions := fsm.Transitions{
		fsm.Tuple(model.PDUSessionInactive, model.InitPduSessionEstablishmentRequestEvent): model.PDUSessionActivePending,

		fsm.Tuple(model.PDUSessionActivePending, model.EstablishmentReject): model.PDUSessionActivePending,
		fsm.Tuple(model.PDUSessionActivePending, model.EstablishmentAccept): model.PDUSessionActive,

		fsm.Tuple(model.PDUSessionActive, model.ModificationRequest): model.PDUModificationPending,
		fsm.Tuple(model.PDUSessionActive, model.ReleaseRequest):      model.PDUSessionInactivePending,
		fsm.Tuple(model.PDUSessionActive, model.ReleaseCommand):      model.PDUSessionInactive,

		fsm.Tuple(model.PDUModificationPending, model.ModificationComplete): model.PDUSessionActive,
		fsm.Tuple(model.PDUModificationPending, model.ModificationReject):   model.PDUSessionInactive,

		fsm.Tuple(model.PDUSessionInactivePending, model.ReleaseCommand): model.PDUSessionInactive,
	}

	callbacks := fsm.Callbacks{
		model.PDUSessionInactive:        sm_PDUSessionInactive,
		model.PDUSessionActivePending:   sm_PDUSessionActivePending,
		model.PDUSessionInactivePending: sm_PDUSessionInactivePending,
		model.PDUSessionActive:          sm_PDUSessionActive,
		model.PDUModificationPending:    sm_PDUModificationPending,
	}

	return fsm.NewFsm(fsm.Options{
		Transitions:           transitions,
		Callbacks:             callbacks,
		GenericCallback:       commonTrigger,
		NonTransitionalEvents: []model.EventType{model.PduSessionInit, model.DestroyPduSession, model.Terminate},
	}, w)
}

func sm_PDUSessionInactive(state *fsm.State, event *fsm.EventData) {
	pdu := fsm.GetStateInfo[PduSession](state)
	ueCtx := pdu.ueCtx
	switch event.Type() {
	case model.EntryEvent:
		pdu.Info("On State 5GSM_PDUSession_Inactive")
	case model.InitPduSessionEstablishmentRequestEvent:
		stats.GlobalStats.StartProcedure(stats.ProcPduEstablish)
		params := fsm.GetEventData[map[string]any](event)
		ueCtx.triggerInitPduSessionRequestInner(pdu, params)
	default:
	}
}
func sm_PDUSessionActivePending(state *fsm.State, event *fsm.EventData) {
	pdu := fsm.GetStateInfo[PduSession](state)
	switch event.Type() {
	case model.EntryEvent:
		pdu.Info("On State 5GSM_PDUSession_Active_Pending")
	case model.EstablishmentAccept:
		stats.GlobalStats.CompleteProcedure(stats.ProcPduEstablish)
		pdu.Info("PDU Session ready")
	case model.EstablishmentReject:
		stats.GlobalStats.FailProcedure(stats.ProcPduEstablish)
	default:
	}
}
func sm_PDUSessionInactivePending(state *fsm.State, event *fsm.EventData) {
	pdu := fsm.GetStateInfo[PduSession](state)
	ueCtx := pdu.ueCtx
	switch event.Type() {
	case model.EntryEvent:
		pdu.Info("On State 5GSM_PDUSession_Inactive_Pending")
	case model.ReleaseCommand:
		ueCtx.triggerInitPduSessionReleaseComplete(pdu)
		ueCtx.deletePduSession(pdu.id)
	default:
	}
}
func sm_PDUSessionActive(state *fsm.State, event *fsm.EventData) {
	pdu := fsm.GetStateInfo[PduSession](state)
	ueCtx := pdu.ueCtx
	switch event.Type() {
	case model.EntryEvent:
		pdu.Info("On State 5GSM_PDUSession_Active")
	case model.ModificationRequest:
	case model.ReleaseRequest:
		ueCtx.triggerInitPduSessionReleaseRequest(pdu)
	case model.ReleaseCommand:
		pdu.Info("[UE][NAS] Successfully released PDU Session from UE Context")
		ueCtx.triggerInitPduSessionReleaseComplete(pdu)
		ueCtx.deletePduSession(pdu.id)
	default:
	}
}
func sm_PDUModificationPending(state *fsm.State, event *fsm.EventData) {
	pdu := fsm.GetStateInfo[PduSession](state)
	// ueCtx := pdu.ueCtx
	switch event.Type() {
	case model.EntryEvent:
		pdu.Info("On State 5GSM_PDUSession_Active")
	case model.ModificationComplete:
	case model.ModificationReject:
	default:
	}
}
