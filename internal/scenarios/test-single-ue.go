package scenarios

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"stormsim/internal/common/ds"
	"stormsim/internal/common/logger"
	"stormsim/internal/common/pool"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/monitoring/oambackend"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"sync"
	"syscall"
	"time"

	"github.com/reogac/utils/oam"
)

var testLogger *logger.Logger

func InitScenarioLogger(
	cfg *config.Config,
	maxPool int,
	nSctpWorker int,
	httpSrv oam.OamServer,
	ctx context.Context,
) {
	testLogger = logger.InitLogger("info", nil)

	nUe := 0
	for _, scenario := range cfg.Scenarios {
		nUe += scenario.NUEs
	}
	nGnb := len(cfg.GNodeBConfig.ListGnbs)

	_ = pool.InitWorkerPool(ctx, maxPool, nSctpWorker, nUe, nGnb)
	uecontext.InitUeContextPool(&cfg.Testing, ctx)

	//logger.RLinkConnStats = make(map[string]*logger.RlinkConnStat, nUe+len(cfg.GNodeBConfig.ListGnbs))

	//logger.SctpConnStats = make(map[string]*logger.SctpConnStat, len(cfg.GNodeBConfig.ListGnbs))

	//logger.TaskStats = make(map[string][]*logger.TaskStat, nUe+len(cfg.GNodeBConfig.ListGnbs))

	// start oam server
	var err error
	if cfg.RemoteServer.Enable {
		name, rootId, getter := oambackend.GetOamBackendInfo(oamApi)
		httpSrv, err = oam.StartOamServer(
			fmt.Sprintf("%s:%d", cfg.RemoteServer.Ip, cfg.RemoteServer.Port),
			name, rootId, getter)

		if err != nil {
			oamLogger.Fatal("Cannot open Remote server: %s", err.Error())
		} else {
			oamLogger.Info("=========== started remote server: %s:%d ==========", cfg.RemoteServer.Ip, cfg.RemoteServer.Port)
		}
	} else {
		oamLogger.Info("=========== not remote server ==========")
	}
}

func TestSingleUE(cfg *config.Config, isReplay bool, replayFile string) {
	testLogger.Info("=================== Simulate only 1 UE ===================")

	var wg sync.WaitGroup
	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	InitScenarioLogger(cfg, 50, 10, nil, ctx)

	// create gnbs
	gnbs := createGnbs(len(cfg.GNodeBConfig.ListGnbs), cfg.GNodeBConfig, cfg.AMFs, cfg.Logging.GnbLogBufferSize, &wg, ctx)
	oamApi.addGnbs(gnbs)
	getGnb := getNextGnb(gnbs)
	gnb, _ := getGnb()

	// worker pool for handling UE
	if replayFile == "" && cfg.Testing.EnableReplay {
		cancel()
		return
	}
	if isReplay { // if isReplay turn-off fuzzer test vs off print state/event to log
		testLogger.Warn("Turn-off Fuzz in mode Replay")
		cfg.Testing.EnableFuzz = false
		cfg.Testing.EnableReplay = true

		var err error
		uecontext.Replay, err = uecontext.LoadFromFile(replayFile)
		if err != nil {
			testLogger.Fatal("Cannot load Replay file: %s: %v", replayFile, err)
			cancel()
			return
		}
	} else {
		cfg.Testing.EnableReplay = false
	}
	uecontext.InitUeContextPool(&cfg.Testing, ctx)

	ueCtx := uecontext.CreateUe(
		cfg.DefaultUe,
		cfg.Logging.UeLogBufferSize,
		0,
		gnb.GetId(),
		cfg.Testing.EnableReplay,
		cfg.Testing.EnableFuzz,
		&wg,
		ctx,
	)
	oamApi.addUes(ueCtx)

	ueTasks := ueCtx.GetEventQueue()

	if !isReplay && !cfg.Testing.EnableFuzz { // if not replay: run custom behaviour
		for _, event := range cfg.Scenarios[0].UeEvents {
			sendTask(&event, ueTasks, nil, 0)
		}
	} else if !isReplay && cfg.Testing.EnableFuzz {
		// if enable fuzzing test: set ue security ctx
		// by excute registration init event for ue
		time.Sleep(time.Second)
		ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.RegisterInit})
	}

	// listen stop signal: Ctrl+C
	go func() {
		<-sigStop
		if cfg.Testing.EnableFuzz {
			now := time.Now()
			tail := now.Format("_150405") // HHMMSS
			logfile := fmt.Sprintf("logger%s.yaml", tail)
			uecontext.Capture.SaveToFile(logfile)
		}
		cancel()
		os.Exit(1)
	}()

	wg.Wait()
	select {}
}

func sendTask(
	event *config.EventInfo,
	ueTasks *ds.Tasks[*uecontext.EventUeData],
	groupEvent chan uecontext.EventUeData,
	idGroup int,
) {
	e := uecontext.EventUeData{EventType: event.Event, Delay: event.TimeBeforeExcuteEvent}

	if e.EventType != model.EntryEvent {
		// testLogger.Info("Send event %s to UE", event.Event)
		// delay process event

		// ================== for only 1 ue: fuzz test, re-produce test =================
		if ueTasks != nil {
			time.Sleep(time.Duration(event.TimeBeforeExcuteEvent) * time.Second)
			testLogger.Info("[Setup Event] Send event [%s] to UE", event.Event)
			ueTasks.AssignTask(&e) // send event/task direct to ue
			return
		}

		// ========================== for multi ue: load test ==========================
		if event.Event == model.XnHandover ||
			event.Event == model.N2Handover {
			time.Sleep(time.Duration(event.TimeBeforeExcuteEvent) * time.Second)
		}
		if groupEvent != nil {
			testLogger.Info("[Setup Event] Send event [%s] to GroupUE-%d", event.Event, idGroup)
			groupEvent <- e // send event to group
		}
	} else {
		testLogger.Error("Cannot send event %s to UE", event.Event)
	}
}

func createGnbs(
	nGnbs int,
	cfg config.GNodeBConfig,
	amfs []model.AMF,
	logBufferSize int,
	wg *sync.WaitGroup,
	ctx context.Context,
) map[string]*gnbcontext.GnbContext {

	gnbs := make(map[string]*gnbcontext.GnbContext, nGnbs)

	for i := range nGnbs {
		currentControlIF := cfg.DefaultControlIF
		currentControlIF.Port += i

		currentDataIF := cfg.DefaultDataIF
		currentDataIF.Port += i

		gnb := gnbcontext.InitGnb(currentControlIF, currentDataIF, cfg.ListGnbs[i], amfs, logBufferSize, wg, ctx)
		gnbs[cfg.ListGnbs[i].GnbId] = gnb
	}

	for _, gnb := range gnbs {
		if !gnb.IsReady() {
			gnb.Fatal("Cannot connect to AMF")
			os.Exit(1)
		} // check gnb success connect to amf
	}
	return gnbs
}
