package scenarios

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"stormsim/internal/common/pool"
	"stormsim/internal/common/stats"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/pkg/config"
	"sync"
	"syscall"
	"time"

	"github.com/reogac/utils/oam"
)

func TestScenarios(cfg *config.Config) {
	var wg sync.WaitGroup
	var httpSrv oam.OamServer
	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	stats.GlobalHistory.StartCron(5 * time.Second)

	go func() {
		<-sigStop
		stats.GlobalHistory.StopCron()
		printFinalStats()

		os.Exit(1)

		// Stop worker pools gracefully
		if pool.WorkerPool != nil {
			pool.WorkerPool.StopAndWait()
		}

		if httpSrv != nil {
			httpSrv.Stop()
			oamLogger.Warn("Close Remote server")
		}
		cancel()
		// oldStats := uecontext.GetStatistics()
		// fmt.Println("Total registered:", oldStats.CountRegistered.Load())
		// fmt.Println("Total fail registration:", oldStats.CountRegisterFail.Load())
		// fmt.Println("Total PDU Session:", oldStats.CountPduSS.Load())
	}()

	InitScenarioLogger(cfg, 0, 0, httpSrv, ctx)
	testLogger.Warn("Cannot use the --tunnel option in multi-ue/gnb scenarios !!!")

	// create initGnbs
	initGnbs := createGnbs(len(cfg.GNodeBConfig.ListGnbs), cfg.GNodeBConfig, cfg.AMFs, cfg.Logging.GnbLogBufferSize, &wg, ctx)
	oamApi.addGnbs(initGnbs)

	// create UEs group with your custom scenarios
	var groupEvents map[int]chan uecontext.EventUeData = make(map[int]chan uecontext.EventUeData)
	for id, scenario := range cfg.Scenarios {
		// load gnb for group
		gnbs := map[string]*gnbcontext.GnbContext{}
		for _, gnbid := range scenario.Gnbs {
			if initGnbs[gnbid] != nil {
				gnbs[gnbid] = initGnbs[gnbid]
			}
		}
		if len(gnbs) == 0 {
			testLogger.Fatal("Group-UE %d cannot find any Gnb %v", id, scenario.Gnbs)
			os.Exit(1)
		}

		// create group UEs
		group := newUeGroup(id, scenario.NUEs, gnbs, cfg.DefaultUe, cfg.Logging.UeLogBufferSize, &wg, ctx)
		groupEvents[group.id] = group.GroupEvents

		cfg.DefaultUe.Msin = incrementMsin(cfg.DefaultUe.Msin, scenario.NUEs)
	}
	// load your custom events
	for id, scenario := range cfg.Scenarios {
		wg.Add(1)
		go func() {
			wg.Done()
			for _, event := range scenario.UeEvents {
				sendTask(&event, nil, groupEvents[id], id)
			}
		}()
	}

	//NOTE: enable if want loop
	// time.Sleep(70 * time.Second)
	// t := time.NewTicker(10 * time.Second)
	// go func() {
	// 	for {
	// 		select {
	// 		case <-t.C:
	// 			for id := range cfg.Scenarios {
	// 				sendTask(&config.EventInfo{
	// 					Event: model.XnHandover,
	// 				}, nil, groupEvents[id], id)
	// 			}
	// 		case <-ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	wg.Wait()
	testLogger.Info("======All testcases finished!======")
}

func printFinalStats() {
	current := stats.GlobalStats.GetSnapshot()
	fmt.Println("\n=== Final Procedure Statistics ===")
	for proc, stat := range current {
		fmt.Printf("%s: completed=%d, failed=%d, running=%d\n",
			proc, stat.Completed, stat.Failed, stat.Running)
	}
}
