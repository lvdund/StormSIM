package scenarios

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"sync"
	"syscall"
	"time"
)

// TestCSVHandover runs handover scenario based on CSV data
func TestCSVHandover(cfg *config.Config, csvCfg config.CSVHandoverConfig) {
	testLogger.Info("=================== CSV Handover Test ==================")
	testLogger.Info("Loading handover events from: %s", csvCfg.FilePath)

	// Load handover events from CSV
	handoverSteps, err := config.LoadHandoverEventsFromCSV(csvCfg)
	if err != nil {
		testLogger.Fatal("Failed to load CSV handover events: %v", err)
		return
	}
	testLogger.Info("Loaded %d handover events from CSV", len(handoverSteps))

	if len(handoverSteps) == 0 {
		testLogger.Warn("No handover events found in CSV file")
		return
	}

	var wg sync.WaitGroup
	sigStop := make(chan os.Signal, 1)
	signal.Notify(sigStop, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	InitScenarioLogger(cfg, 50, 10, nil, ctx)

	// Create gNBs
	gnbs := createGnbs(len(cfg.GNodeBConfig.ListGnbs), cfg.GNodeBConfig, cfg.AMFs, cfg.Logging.GnbLogBufferSize, &wg, ctx)

	// Create gNB lookup map for handover
	// Find initial gNB (first gNB in config or from first CSV step)
	var initialGnb *gnbcontext.GnbContext
	gnbMap := make(map[string]*gnbcontext.GnbContext)
	for id, gnb := range gnbs {
		if initialGnb == nil {
			initialGnb = gnb
		}
		gnbMap[id] = gnb
	}

	if initialGnb == nil {
		testLogger.Fatal("No gNB available for initial connection")
		cancel()
		return
	}

	// Create single UE
	testLogger.Info("Creating UE with initial gNB: %s", initialGnb.GetId())
	ueCtx := uecontext.CreateUe(
		cfg.DefaultUe,
		cfg.Logging.UeLogBufferSize,
		0,
		initialGnb.GetId(),
		false,
		false,
		&wg,
		ctx,
	)
	ueTasks := ueCtx.GetEventQueue()

	// Create handover tracker
	tracker := NewHandoverTracker()

	// Register UE first
	testLogger.Info("Step 0: Registering UE...")
	ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.RegisterInit})
	time.Sleep(2 * time.Second) // Wait for registration

	// Establish PDU session (required for handover)
	testLogger.Info("Step 1: Establishing PDU session...")
	ueTasks.AssignTask(&uecontext.EventUeData{EventType: model.PduSessionInit})
	time.Sleep(2 * time.Second) // Wait for PDU session

	// Track current gNB
	currentGnb := initialGnb

	// Listen for stop signal
	go func() {
		<-sigStop
		testLogger.Info("\n" + tracker.PrintSummary())
		printDetailedResults(tracker)
		cancel()
		os.Exit(0)
	}()

	// Execute handover events from CSV
	testLogger.Info("Starting CSV handover sequence...")
	stepDelay := time.Duration(csvCfg.StepDelayMs) * time.Millisecond
	if stepDelay == 0 {
		stepDelay = 1 * time.Second // Default 1 second between handovers
	}

	for _, step := range handoverSteps {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Find target gNB
		targetGnb, ok := gnbMap[step.ToGnbId]
		if !ok {
			testLogger.Error("Step %d: Target gNB %s not found in config, skipping", step.Step, step.ToGnbId)
			continue
		}

		// Skip if current and target are same
		if currentGnb.GetId() == targetGnb.GetId() {
			testLogger.Warn("Step %d: Source and target gNB are same (%s), skipping", step.Step, step.ToGnbId)
			continue
		}

		hoTypeStr := "Xn"
		if step.IsN2Handover() {
			hoTypeStr = "N2"
		}
		testLogger.Info("Step %d: %s Handover from %s to %s", step.Step, hoTypeStr, currentGnb.GetId(), targetGnb.GetId())

		// Record start
		tracker.StartHandover(step.Step, currentGnb.GetId(), targetGnb.GetId(), step.HandoverType)

		// Trigger handover based on type
		triggerCSVHandover(currentGnb, targetGnb, ueCtx, tracker, step.Step, step.IsXnHandover())
		currentGnb = targetGnb

		// Wait between steps
		time.Sleep(stepDelay)
	}

	// Print final summary
	testLogger.Info("CSV handover sequence completed")
	testLogger.Info(tracker.PrintSummary())
	printDetailedResults(tracker)

	wg.Wait()
}

// triggers handover for CSV scenario
func triggerCSVHandover(
	oldGnb, newGnb *gnbcontext.GnbContext,
	ueCtx *uecontext.UeContext,
	tracker *HandoverTracker,
	step int,
	isXn bool,
) {
	ueId := ueCtx.GetId()
	if isXn {
		gnbcontext.TriggerXnHandover(oldGnb, newGnb, ueId)
	} else {
		gnbcontext.TriggerNgapHandover(oldGnb, newGnb, ueId)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-ticker.C:
			if newGnb.IsHandoverSuccess(ueId) {
				if ueCtx.GetMMState().CurrentState() == model.Registered {
					tracker.CompleteHandover(step, true, "")
					testLogger.Info("Step %d: Xn Handover SUCCESS", step)
				} else {
					tracker.CompleteHandover(step, false, "UE not in Registered state")
					testLogger.Warn("Step %d: Xn Handover FAILED", step)
				}
				return
			}
		case <-timeout:
			ticker.Stop()
			return
		}
	}
}

// printDetailedResults prints detailed handover results
func printDetailedResults(tracker *HandoverTracker) {
	results := tracker.GetResults()
	if len(results) == 0 {
		return
	}

	fmt.Println("\n========== Detailed Handover Results ==========")
	fmt.Printf("%-6s %-10s %-10s %-6s %-10s %-12s %s\n",
		"Step", "From", "To", "Type", "Status", "Duration", "Reason")
	fmt.Println("---------------------------------------------------------------")

	for _, r := range results {
		hoType := "Xn"
		if r.HandoverType == 2 {
			hoType = "N2"
		}
		status := "SUCCESS"
		if !r.Success {
			status = "FAILED"
		}
		fmt.Printf("%-6d %-10s %-10s %-6s %-10s %-12s %s\n",
			r.Step, r.FromGnb, r.ToGnb, hoType, status, r.Duration, r.FailureReason)
	}
	fmt.Println("================================================")
}
