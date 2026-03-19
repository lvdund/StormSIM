package scenarios

import (
	"fmt"
	"sync"
	"time"
)

// HandoverResult tracks the outcome of a single handover operation
type HandoverResult struct {
	Step          int
	FromGnb       string
	ToGnb         string
	HandoverType  int // 1=Xn, 2=N2
	Success       bool
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	FailureReason string // Empty if success
}

// HandoverTracker monitors and records handover results
type HandoverTracker struct {
	Results      []HandoverResult
	mu           sync.RWMutex
	totalCount   int
	successCount int
	failCount    int
}

// NewHandoverTracker creates a new handover tracker
func NewHandoverTracker() *HandoverTracker {
	return &HandoverTracker{
		Results: []HandoverResult{},
	}
}

// StartHandover records the start of a handover operation
func (ht *HandoverTracker) StartHandover(step int, fromGnb, toGnb string, hoType int) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	result := HandoverResult{
		Step:         step,
		FromGnb:      fromGnb,
		ToGnb:        toGnb,
		HandoverType: hoType,
		StartTime:    time.Now(),
		Success:      false, // Will be updated on completion
	}
	ht.Results = append(ht.Results, result)
	ht.totalCount++
}

// CompleteHandover marks a handover as successfully completed
func (ht *HandoverTracker) CompleteHandover(step int, success bool, failureReason string) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	for i := len(ht.Results) - 1; i >= 0; i-- {
		if ht.Results[i].Step == step && ht.Results[i].EndTime.IsZero() {
			ht.Results[i].EndTime = time.Now()
			ht.Results[i].Duration = ht.Results[i].EndTime.Sub(ht.Results[i].StartTime)
			ht.Results[i].Success = success
			ht.Results[i].FailureReason = failureReason

			if success {
				ht.successCount++
			} else {
				ht.failCount++
			}
			return
		}
	}
}

// GetStats returns summary statistics
func (ht *HandoverTracker) GetStats() (total, success, fail int) {
	ht.mu.RLock()
	defer ht.mu.RUnlock()
	return ht.totalCount, ht.successCount, ht.failCount
}

// GetResults returns a copy of all results
func (ht *HandoverTracker) GetResults() []HandoverResult {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	results := make([]HandoverResult, len(ht.Results))
	copy(results, ht.Results)
	return results
}

// GetSuccessRate returns the handover success rate as a percentage
func (ht *HandoverTracker) GetSuccessRate() float64 {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	if ht.totalCount == 0 {
		return 0.0
	}
	return float64(ht.successCount) / float64(ht.totalCount) * 100.0
}

// PrintSummary returns a summary string of handover results
func (ht *HandoverTracker) PrintSummary() string {
	total, success, fail := ht.GetStats()
	rate := ht.GetSuccessRate()

	return fmt.Sprintf(
		"Handover Summary: Total=%d, Success=%d, Failed=%d, Rate=%.1f%%",
		total, success, fail, rate,
	)
}
