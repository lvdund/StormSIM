package uecontext

import (
	"context"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/logger"
	"stormsim/internal/common/pool"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
)

var fsmLogger *logger.Logger

func init() {
	fsmLogger = logger.InitLogger("info", map[string]string{"mod": "fsm"})
}

type ueContextPool struct {
	fsm_mm fsm.StateMachine
	fsm_sm fsm.StateMachine
	timer  timer.Timer
}

var _uePool *ueContextPool

func InitUeContextPool(fuzzOptions *config.TestingConf, ctx context.Context) {
	if _uePool == nil {
		_uePool = &ueContextPool{}

		fsm_mm := initFSM(pool.MmWorkerPool)
		fsm_sm := initPduFSM(pool.SmWorkerPool)
		initUeFsm(fuzzOptions, fsm_mm, fsm_sm, ctx)
	}
}

func initUeFsm(fuzzOptions *config.TestingConf, fsm_mm, fsm_sm *fsm.Fsm, ctx context.Context) {
	if fuzzOptions.EnableFuzz {
		fsmLogger.Info("[FSM] Enable Fuzzing Test")
		var fuzz_mm fsm.FuzzerOptions = fsm.FuzzerOptions{
			FuzzMode:       false,
			PossibleStates: []model.StateType{},
			PossibleEvents: []model.EventType{},
		}
		var fuzz_sm fsm.FuzzerOptions = fsm.FuzzerOptions{
			FuzzMode:       true,
			PossibleStates: []model.StateType{},
			PossibleEvents: []model.EventType{},
		}

		fuzz_mm.PossibleStates = append(fuzz_mm.PossibleStates, fuzzOptions.Mm.States...)
		fuzz_mm.PossibleEvents = append(fuzz_mm.PossibleEvents, fuzzOptions.Mm.Events...)
		fuzz_sm.PossibleEvents = append(fuzz_sm.PossibleEvents, fuzzOptions.Sm.Events...)
		fuzz_sm.PossibleEvents = append(fuzz_sm.PossibleEvents, fuzzOptions.Sm.Events...)

		_uePool.fsm_mm = fsm.NewFsmFuzzer(fsm_mm, &fuzz_mm, ctx)
		_uePool.fsm_sm = fsm.NewFsmFuzzer(fsm_sm, &fuzz_sm, ctx)
	} else {
		_uePool.fsm_mm = fsm_mm
		_uePool.fsm_sm = fsm_sm
	}
}

// CleanupUeContextPool cleans up the global UE context pool
func CleanupUeContextPool() {
	if _uePool != nil {
		_uePool = nil
	}
}

type EventUeData struct {
	Ue        *UeContext
	EventType model.EventType // external trigger
	Msg       []byte
	Delay     uint8
	Params    map[string]any
}
