package uecontext

import (
	"fmt"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/stats"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/pkg/model"
	"sync/atomic"
	"time"

	"github.com/alitto/pond/v2"
)

// UEStatistics holds global UE registration statistics
type UEStatistics struct {
	CountRegistered   atomic.Int32
	CountRegisterFail atomic.Int32
	CountPduSS        atomic.Int32
}

var globalStats = &UEStatistics{}

// GetStatistics returns the global UE statistics instance
func GetStatistics() *UEStatistics {
	return globalStats
}

func initFSM(w pond.Pool) *fsm.Fsm {
	transitions := fsm.Transitions{
		fsm.Tuple(model.Deregistered, model.InitRegistrationRequestEvent): model.RegisteredInitiated,
		fsm.Tuple(model.Deregistered, model.T3502Event):                   model.RegisteredInitiated,
		fsm.Tuple(model.Deregistered, model.T3511Event):                   model.RegisteredInitiated,

		fsm.Tuple(model.RegisteredInitiated, model.RegistrationAcceptEvent): model.Registered,
		fsm.Tuple(model.RegisteredInitiated, model.RegistrationRejectEvent): model.Deregistered,
		fsm.Tuple(model.RegisteredInitiated, model.AuthFailEvent):           model.Deregistered,
		fsm.Tuple(model.RegisteredInitiated, model.SecurityModeFailEvent):   model.Deregistered,
		fsm.Tuple(model.RegisteredInitiated, model.MissingInfo):             model.Deregistered,
		fsm.Tuple(model.RegisteredInitiated, model.T3511Event):              model.Deregistered,

		fsm.Tuple(model.Registered, model.InitDeregistrationRequestEvent):    model.Deregistered,
		fsm.Tuple(model.Registered, model.NetworkDeregistrationRequestEvent): model.Deregistered,
	}

	callbacks := fsm.Callbacks{
		model.Deregistered:        mm_Deregistered,
		model.RegisteredInitiated: mm_RegisteredInitiated,
		model.Registered:          mm_Registered,
	}

	return fsm.NewFsm(fsm.Options{
		Transitions:     transitions,
		Callbacks:       callbacks,
		GenericCallback: commonTrigger,
		NonTransitionalEvents: []model.EventType{
			model.GmmMessageEvent,
			model.RegisterInit,
			model.DeregistraterInit,
			model.XnHandover,
			model.N2Handover,
			model.PduSessionInit,
			model.IdleInit,
			model.ServiceRequestInit,
			model.DestroyPduSession,
			model.Terminate,
			model.NullInit,
		},
	}, w)
}

func mm_Deregistered(state *fsm.State, event *fsm.EventData) {
	ueCtx := fsm.GetStateInfo[UeContext](state)
	switch event.Type() {
	case model.EntryEvent:
		ueCtx.Info("On State 5GMM_Deregisterd")
	case model.InitRegistrationRequestEvent:
		stats.GlobalStats.StartProcedure(stats.ProcRegistration)
		if ueCtx.timerEngine.BlockTimerEvent {
			ueCtx.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.T3502Event))
		} else if err := ueCtx.triggerInitRegistration(); err != nil {
			stats.GlobalStats.FailProcedure(stats.ProcRegistration)
			ueCtx.Error("Error encoding registration request: %v", err)
			ueCtx.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.MissingInfo))
		}

		// start timer t3510
		// if timeout (no resp or reject from AMF), start t3511
		// after retry 5 times, block registration procedure with t3502
		// t3502 block registration-action
		// if exist, _, _ := ueCtx.timerEngine.GetTimerStatus(timer.T3510); !exist {
		// 	ueCtx.timerEngine.CreateTimer(timer.TimerConfig{
		// 		TimerType:   timer.T3510,
		// 		Duration:    timer.T3510_duration,
		// 		CountMax:    1,
		// 		TimeoutFunc: nil,
		// 		ExpireFunc: func() {
		// 			if err := ueCtx.timerEngine.Start(timer.T3511); err != nil {
		// 				ueCtx.Warn("Expire T3510, cannot create T3511: %v", err)
		// 			} else {
		// 				ueCtx.Info("Expire T3510, start T3511")
		// 			}
		// 		},
		// 	})
		// 	ueCtx.Info("Start Timer T3510")
		// 	ueCtx.timerEngine.Start(timer.T3510)
		// }
		// if exist, _, _ := ueCtx.timerEngine.GetTimerStatus(timer.T3511); !exist {
		// 	ueCtx.timerEngine.CreateTimer(timer.TimerConfig{
		// 		TimerType: timer.T3511,
		// 		Duration:  timer.T3511_duration,
		// 		CountMax:  5,
		// 		TimeoutFunc: func() {
		// 			ueCtx.sendEventMm(fsm.NewEmptyEventData(model.T3511Event))
		// 		},
		// 		ExpireFunc: func() {
		// 			if err := ueCtx.timerEngine.Start(timer.T3502); err != nil {
		// 				ueCtx.Warn("Expire T3511, cannot start T3502: %v", err)
		// 			} else {
		// 				ueCtx.timerEngine.BlockTimerEvent = true
		// 				ueCtx.Info("Expire T3511, start T3502")
		// 			}
		// 		},
		// 	})
		// }
		// if exist, _, _ := ueCtx.timerEngine.GetTimerStatus(timer.T3502); !exist {
		// 	ueCtx.timerEngine.CreateTimer(timer.TimerConfig{
		// 		TimerType: timer.T3502,
		// 		Duration:  timer.T3502_duration,
		// 		CountMax:  1,
		// 		ExpireFunc: func() {
		// 			ueCtx.Info("Timer T3502")
		// 			ueCtx.timerEngine.BlockTimerEvent = false
		// 		},
		// 	})
		// }

	case model.T3511Event:
		ueCtx.Warn("Re-send Registration Request: cause of Timer T3511")
		ueCtx.sendNas(ueCtx.nasPdu)
	case model.T3502Event:
		ueCtx.Warn("Block Registration Reqest cause of Timer T3502")
	default:
	}
}
func mm_RegisteredInitiated(state *fsm.State, event *fsm.EventData) {
	ueCtx := fsm.GetStateInfo[UeContext](state)
	switch event.Type() {
	case model.EntryEvent:
		ueCtx.Info("On State 5GMM_RegisteredInitiated")
	case model.RegistrationAcceptEvent:
		stats.GlobalStats.CompleteProcedure(stats.ProcRegistration)
		ueCtx.timerEngine.RemoveTimer(timer.T3502)
		ueCtx.timerEngine.RemoveTimer(timer.T3510)
		ueCtx.timerEngine.RemoveTimer(timer.T3511)
		if ueCtx.enableFuzz {
			if mm, ok := _uePool.fsm_mm.(*fsm.FsmFuzzer); ok {
				mm.FuzzMode = true
				go mm.AutoRandomEvent(state, event, 2*time.Second)
				ueCtx.Info("======== Start Fuzzing Test ========")
			}
		}
	case model.RegistrationRejectEvent, model.AuthFailEvent, model.SecurityModeFailEvent, model.MissingInfo:
		stats.GlobalStats.FailProcedure(stats.ProcRegistration)
		if exist, _, err := ueCtx.timerEngine.GetTimerStatus(timer.T3510); exist && err == nil {
			ueCtx.timerEngine.StopWithExpireFunc(timer.T3510)
		} else if exist, _, err = ueCtx.timerEngine.GetTimerStatus(timer.T3511); exist && err == nil {
			ueCtx.timerEngine.TriggerTimeout(timer.T3511)
		}
		ueCtx.Info("On State 5GMM_Registered:")
	default:
	}
}
func mm_Registered(state *fsm.State, event *fsm.EventData) {
	ueCtx := fsm.GetStateInfo[UeContext](state)
	switch event.Type() {
	case model.EntryEvent:
		count := globalStats.CountRegistered.Add(1)
		ueCtx.Info("On State 5GMM_Registered: %d", count)
		fmt.Println("On State 5GMM_Registered:", count)
	case model.InitDeregistrationRequestEvent:
		ueCtx.Info("On State 5GMM_DeregistrationInitiated")
		var params map[string]int = map[string]int{"type": 0}
		if fsm.GetEventData[map[string]int](event) != nil {
			params = *fsm.GetEventData[map[string]int](event)
		}
		ueCtx.triggerInitDeregistration(params["type"])
	case model.NetworkDeregistrationRequestEvent:
		ueCtx.Terminate()
	default:
	}
}

func commonTrigger(state *fsm.State, event *fsm.EventData) {
	ueCtx := fsm.GetStateInfo[UeContext](state)

	switch event.Type() {

	// handle 5gmm NAS message from AMF
	case model.GmmMessageEvent:
		nasMsg := fsm.GetEventData[[]byte](event)
		ueCtx.handleNasMsg(*nasMsg)

	// common event
	case model.DeregistrationAcceptEvent:
		ueCtx.state_mm.ForceSetState(model.Deregistered)
		ueCtx.Terminate()

	// handle remote event
	case model.RegisterInit:
		if ueCtx.state_mm.CurrentState() == model.Deregistered {
			ueCtx.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.InitRegistrationRequestEvent))
		} else {
			ueCtx.Warn("Ue is on Registering")
		}
	case model.PduSessionInit:
		if ueCtx.state_mm.CurrentState() == model.Registered {
			params := fsm.GetEventData[map[string]any](event)
			ueCtx.triggerInitPduSessionRequest(params)
		} else {
			ueCtx.Error("Cannot create PDU session: state[%s] UE != Registerd",
				ueCtx.state_mm.CurrentState())
		}
	case model.DestroyPduSession:
		if ueCtx.state_mm.CurrentState() == model.Registered {
			deleteALlPduSession(ueCtx, false)
		} else if ueCtx.state_mm.CurrentState() == model.Deregistered {
			deleteALlPduSession(ueCtx, true)
		}
	case model.IdleInit:
		ueCtx.triggerSwitchToIdle()
		ueCtx.createDRX(25 * time.Millisecond)
	case model.ServiceRequestInit:
		ueCtx.stopDRX()
		ueCtx.initConn()
		if ueCtx.guti != nil {
			ueCtx.triggerInitServiceRequest()
		} else {
			//FIX: ensure that
			// If AMF did not assign us a GUTI, we have to fallback to the usual
			// Registration/Authentification process PDU Sessions will still be recovered
			ueCtx.state_mm.SetNextEvent(fsm.NewEmptyEventData(model.RegisterInit))
		}
	case model.Terminate, model.DeregistraterInit:
		if ueCtx.state_mm.CurrentState() == model.Deregistered {
			ueCtx.Warn("ue[%d] already deregisterd", ueCtx.id)
		} else {
			if ueCtx.state_mm.CurrentState() == model.Registered {
				deleteALlPduSession(ueCtx, false)
			}

			params := fsm.GetEventData[map[string]int](event)
			ueCtx.state_mm.SetNextEvent(fsm.NewEventData(model.InitDeregistrationRequestEvent, params))
		}
	default:
		ueCtx.Error("Unknow event")
	}
}

func deleteALlPduSession(ue *UeContext, local_del bool) {
	ue.Warn("Destroy all Pdu Session - ue[%d]", ue.id)
	if local_del {
		for i := uint8(1); i <= 16; i++ {
			pduSession, _ := ue.getPduSession(i)
			if pduSession != nil && <-pduSession.wait {
				ue.deletePduSession(pduSession.id)
			}
		}
	} else {
		for i := uint8(1); i <= 16; i++ {
			pduSession, _ := ue.getPduSession(i)
			if pduSession != nil {
				pduSession.SendEventSm(fsm.NewEmptyEventData(model.ReleaseRequest))
			}
		}
	}
}
