package stats

import (
	"sync"
	"sync/atomic"
)

type ProcedureType string

const (
	ProcRegistration   ProcedureType = "registration"
	ProcDeregistration ProcedureType = "deregistration"
	ProcPduEstablish   ProcedureType = "pdu_establish"
)

type ProcedureStat struct {
	Running   atomic.Int64
	Completed atomic.Int64
	Failed    atomic.Int64
}

type StatSnapshot struct {
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
}

type StatsCollector struct {
	stats map[ProcedureType]*ProcedureStat
	mu    sync.RWMutex
}

var GlobalStats = NewStatsCollector()

func NewStatsCollector() *StatsCollector {
	sc := &StatsCollector{
		stats: make(map[ProcedureType]*ProcedureStat),
	}
	for _, proc := range []ProcedureType{
		ProcRegistration,
		ProcDeregistration,
		ProcPduEstablish,
	} {
		sc.stats[proc] = &ProcedureStat{}
	}
	return sc
}

func (sc *StatsCollector) StartProcedure(proc ProcedureType) {
	if stat, ok := sc.stats[proc]; ok {
		stat.Running.Add(1)
	}
}

func (sc *StatsCollector) CompleteProcedure(proc ProcedureType) {
	if stat, ok := sc.stats[proc]; ok {
		stat.Running.Add(-1)
		stat.Completed.Add(1)
	}
}

func (sc *StatsCollector) FailProcedure(proc ProcedureType) {
	if stat, ok := sc.stats[proc]; ok {
		stat.Running.Add(-1)
		stat.Failed.Add(1)
	}
}

func (sc *StatsCollector) GetSnapshot() map[ProcedureType]StatSnapshot {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make(map[ProcedureType]StatSnapshot)
	for proc, stat := range sc.stats {
		result[proc] = StatSnapshot{
			Running:   stat.Running.Load(),
			Completed: stat.Completed.Load(),
			Failed:    stat.Failed.Load(),
		}
	}
	return result
}

func (sc *StatsCollector) AddProcedureType(proc ProcedureType) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if _, exists := sc.stats[proc]; !exists {
		sc.stats[proc] = &ProcedureStat{}
	}
}
