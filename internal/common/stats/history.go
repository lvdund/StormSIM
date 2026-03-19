package stats

import (
	"sync"
	"time"
)

const MaxSnapshots = 3600

type HistoricalSnapshot struct {
	Timestamp time.Time                      `json:"timestamp"`
	Stats     map[ProcedureType]StatSnapshot `json:"stats"`
}

type StatsHistory struct {
	snapshots []HistoricalSnapshot
	ticker    *time.Ticker
	stopCh    chan struct{}
	mu        sync.RWMutex
	running   bool
}

var GlobalHistory = NewStatsHistory()

func NewStatsHistory() *StatsHistory {
	return &StatsHistory{
		snapshots: make([]HistoricalSnapshot, 0, MaxSnapshots),
		stopCh:    make(chan struct{}),
	}
}

func (sh *StatsHistory) StartCron(interval time.Duration) {
	sh.mu.Lock()
	if sh.running {
		sh.mu.Unlock()
		return
	}
	sh.running = true
	sh.ticker = time.NewTicker(interval)
	sh.mu.Unlock()

	go func() {
		for {
			select {
			case <-sh.ticker.C:
				sh.takeSnapshot()
			case <-sh.stopCh:
				sh.ticker.Stop()
				return
			}
		}
	}()
}

func (sh *StatsHistory) StopCron() {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.running {
		close(sh.stopCh)
		sh.running = false
	}
}

func (sh *StatsHistory) takeSnapshot() {
	snapshot := HistoricalSnapshot{
		Timestamp: time.Now(),
		Stats:     GlobalStats.GetSnapshot(),
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.snapshots = append(sh.snapshots, snapshot)
	if len(sh.snapshots) > MaxSnapshots {
		sh.snapshots = sh.snapshots[1:]
	}
}

func (sh *StatsHistory) GetAllSnapshots() []HistoricalSnapshot {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	result := make([]HistoricalSnapshot, len(sh.snapshots))
	copy(result, sh.snapshots)
	return result
}

func (sh *StatsHistory) GetSnapshotsSince(since time.Time) []HistoricalSnapshot {
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	result := make([]HistoricalSnapshot, 0)
	for _, snap := range sh.snapshots {
		if snap.Timestamp.After(since) {
			result = append(result, snap)
		}
	}
	return result
}

func (sh *StatsHistory) GetSnapshotCount() int {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return len(sh.snapshots)
}
