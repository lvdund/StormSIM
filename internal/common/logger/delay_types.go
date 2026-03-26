package logger

import (
	"math"
	"sync"
	"time"
)

type StatsResult struct {
	Min    float64 `json:"min"`    // Minimum delay in milliseconds
	Max    float64 `json:"max"`    // Maximum delay in milliseconds
	Mean   float64 `json:"mean"`   // Mean delay in milliseconds
	StdDev float64 `json:"stdDev"` // Standard deviation in milliseconds
	Count  int     `json:"count"`  // Number of measurements
}

// DelayEntry represents a single request-response delay measurement
type DelayEntry struct {
	Protocol     string    `json:"protocol"`     // "nas", "ngap", "rlink"
	RequestType  string    `json:"requestType"`  // Human-readable name of request message
	ResponseType string    `json:"responseType"` // Human-readable name of response message
	SendTime     time.Time `json:"sendTime"`     // When request was sent
	DelayMs      float64   `json:"delayMs"`      // Round-trip delay in milliseconds
}

// ProcedureStats contains statistics for a specific 5G procedure
type ProcedureStats struct {
	Name string `json:"name"`
	StatsResult
	Last float64 `json:"last"` // Duration of the most recent execution
}

// NasPairStats contains statistics for a specific Request-Response pair
type NasPairStats struct {
	Request  string `json:"request"`
	Response string `json:"response"`
	StatsResult
}

// DelayStats contains aggregated delay statistics
type DelayStats struct {
	StatsResult

	// Procedures contains statistics for specific 5G procedures
	Procedures map[string]ProcedureStats `json:"procedures"`

	// NasPairs contains statistics for specific NAS message pairs
	NasPairs []NasPairStats `json:"nasPairs"`
}

// GroupDelayStats contains group-level delay statistics
type GroupDelayStats struct {
	Mean    float64 `json:"mean"`    // Mean delay across all entities
	StdDev  float64 `json:"stdDev"`  // Standard deviation across all entities
	Count   int     `json:"count"`   // Total number of measurements
	UeCount int     `json:"ueCount"` // Number of entities with measurements

	// Procedures contains aggregated procedure statistics
	Procedures map[string]ProcedureStats `json:"procedures"`

	// NasPairs contains aggregated statistics for NAS message pairs
	NasPairs []NasPairStats `json:"nasPairs"`
}

// Procedure definitions
var (
	procStartEvents = map[string]string{
		"RegistrationRequest":            "registration",
		"DeregistrationRequestFromUE":    "deregistration",
		"PduSessionEstablishmentRequest": "pdu_establish",
		"PduSessionReleaseRequest":       "pdu_release",
	}

	procEndEvents = map[string]struct {
		Proc string
		Type string // "send" or "receive"
	}{
		"RegistrationComplete":          {"registration", "send"},
		"DeregistrationAcceptFromUE":    {"deregistration", "receive"},
		"PduSessionEstablishmentAccept": {"pdu_establish", "receive"},
		"PduSessionReleaseComplete":     {"pdu_release", "send"},
	}
)

type pendingEntry struct {
	SendTime   time.Time
	LogIndex   int
	Generation uint64
}

type DelayTracker struct {
	buffer  *RingBuffer[DelayEntry]
	pending map[string]map[string]pendingEntry

	procStarts  map[string]time.Time
	procHistory map[string][]float64

	mu sync.RWMutex
}

func NewDelayTracker(capacity int) *DelayTracker {
	if capacity <= 0 {
		capacity = 100
	}
	return &DelayTracker{
		buffer: NewRingBuffer[DelayEntry](capacity),
		pending: map[string]map[string]pendingEntry{
			"nas":   {},
			"ngap":  {},
			"rlink": {},
		},
		procStarts:  make(map[string]time.Time),
		procHistory: make(map[string][]float64),
	}
}

// trackProcedureStart records the start time of a procedure
func (dt *DelayTracker) trackProcedureStart(msgType string) {
	if proc, ok := procStartEvents[msgType]; ok {
		dt.procStarts[proc] = time.Now()
	}
}

// trackProcedureEnd records the end time of a procedure and stores the duration
func (dt *DelayTracker) trackProcedureEnd(msgType, eventType string) {
	if def, ok := procEndEvents[msgType]; ok {
		if def.Type == eventType {
			if start, ok := dt.procStarts[def.Proc]; ok {
				duration := time.Since(start).Seconds() * 1000
				dt.procHistory[def.Proc] = append(dt.procHistory[def.Proc], duration)
				delete(dt.procStarts, def.Proc)
			}
		}
	}
}

func (dt *DelayTracker) RecordSend(protocol, msgType string) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.trackProcedureStart(msgType)
	dt.trackProcedureEnd(msgType, "send")

	if dt.pending[protocol] == nil {
		dt.pending[protocol] = make(map[string]pendingEntry)
	}

	now := time.Now()
	entry := DelayEntry{
		Protocol:     protocol,
		RequestType:  msgType,
		ResponseType: "Unknown",
		SendTime:     now,
		DelayMs:      0,
	}
	idx, gen := dt.buffer.Push(entry)
	dt.pending[protocol][msgType] = pendingEntry{
		SendTime:   now,
		LogIndex:   idx,
		Generation: gen,
	}
}

func (dt *DelayTracker) RecordReceive(protocol, responseType string) {
	if responseType == "DlNasTransport" {
		return
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.trackProcedureEnd(responseType, "receive")

	pendingForProtocol := dt.pending[protocol]
	if len(pendingForProtocol) == 0 {
		entry := DelayEntry{
			Protocol:     protocol,
			RequestType:  "Unknown",
			ResponseType: responseType,
			SendTime:     time.Time{},
			DelayMs:      0,
		}
		dt.buffer.Push(entry)
		return
	}

	var matchedType string
	var matchedPending pendingEntry

	if protocol == "nas" {
		if validRequests, ok := NasResponseToRequests[responseType]; ok {
			for _, reqType := range validRequests {
				if pending, exists := pendingForProtocol[reqType]; exists {
					if matchedType == "" || pending.SendTime.Before(matchedPending.SendTime) {
						matchedType = reqType
						matchedPending = pending
					}
				}
			}
			if matchedType == "" {
				for msgType, pending := range pendingForProtocol {
					if matchedType == "" || pending.SendTime.Before(matchedPending.SendTime) {
						matchedType = msgType
						matchedPending = pending
					}
				}
				if matchedType != "" {
					delay := time.Since(matchedPending.SendTime).Seconds() * 1000
					entry := DelayEntry{
						Protocol:     protocol,
						RequestType:  "Unknown",
						ResponseType: responseType,
						SendTime:     matchedPending.SendTime,
						DelayMs:      delay,
					}
					dt.buffer.Update(matchedPending.LogIndex, matchedPending.Generation, entry)
					delete(pendingForProtocol, matchedType)
					return
				}
				entry := DelayEntry{
					Protocol:     protocol,
					RequestType:  "Unknown",
					ResponseType: responseType,
					SendTime:     time.Time{},
					DelayMs:      0,
				}
				dt.buffer.Push(entry)
				return
			}
		}
	}

	if matchedType == "" {
		for msgType, pending := range pendingForProtocol {
			if matchedType == "" || pending.SendTime.Before(matchedPending.SendTime) {
				matchedType = msgType
				matchedPending = pending
			}
		}
	}

	if matchedType == "" {
		return
	}

	delay := time.Since(matchedPending.SendTime).Seconds() * 1000
	entry := DelayEntry{
		Protocol:     protocol,
		RequestType:  matchedType,
		ResponseType: responseType,
		SendTime:     matchedPending.SendTime,
		DelayMs:      delay,
	}
	dt.buffer.Update(matchedPending.LogIndex, matchedPending.Generation, entry)
	delete(pendingForProtocol, matchedType)
}

// GetLogs returns the last n delay entries (most recent first), or all if n <= 0
func (dt *DelayTracker) GetLogs(last int) []DelayEntry {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	if last <= 0 {
		return dt.buffer.GetAll()
	}
	return dt.buffer.GetLast(last)
}

// GetLogsByProtocol returns delay entries filtered by protocol
func (dt *DelayTracker) GetLogsByProtocol(protocol string, last int) []DelayEntry {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	all := dt.buffer.GetAll()
	if protocol == "" || protocol == "all" {
		if last <= 0 || last >= len(all) {
			return all
		}
		// Return last n entries (most recent = end of array)
		return all[len(all)-last:]
	}

	// Filter by protocol
	filtered := make([]DelayEntry, 0)
	for _, entry := range all {
		if entry.Protocol == protocol {
			filtered = append(filtered, entry)
		}
	}

	if last <= 0 || last >= len(filtered) {
		return filtered
	}
	return filtered[len(filtered)-last:]
}

// GetStats computes delay statistics from all entries
func (dt *DelayTracker) GetStats() DelayStats {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	entries := dt.buffer.GetAll()
	if len(entries) == 0 {
		return DelayStats{}
	}

	delays := make([]float64, 0, len(entries))
	for _, entry := range entries {
		delays = append(delays, entry.DelayMs)
	}

	stats := calculateStats(delays)

	return DelayStats{
		StatsResult: stats,
		Procedures:  dt.getProcedureStats(),
		NasPairs:    dt.getNasPairStats(entries),
	}
}

// GetStatsByProtocol computes delay statistics for a specific protocol
func (dt *DelayTracker) GetStatsByProtocol(protocol string) DelayStats {
	entries := dt.GetLogsByProtocol(protocol, 0)
	if len(entries) == 0 {
		return DelayStats{
			Procedures: make(map[string]ProcedureStats),
			NasPairs:   make([]NasPairStats, 0),
		}
	}

	delays := make([]float64, 0, len(entries))
	for _, entry := range entries {
		delays = append(delays, entry.DelayMs)
	}

	stats := calculateStats(delays)

	return DelayStats{
		StatsResult: stats,
		Procedures:  make(map[string]ProcedureStats),
		NasPairs:    dt.getNasPairStats(entries),
	}
}

// getNasPairStats calculates statistics grouped by Request/Response pair
func (dt *DelayTracker) getNasPairStats(entries []DelayEntry) []NasPairStats {
	groups := make(map[string][]float64)
	keys := make(map[string]struct{ req, resp string })

	for _, e := range entries {
		k := e.RequestType + "|" + e.ResponseType
		groups[k] = append(groups[k], e.DelayMs)
		keys[k] = struct{ req, resp string }{e.RequestType, e.ResponseType}
	}

	stats := make([]NasPairStats, 0, len(groups))
	for k, delays := range groups {
		if len(delays) == 0 {
			continue
		}

		pairStats := calculateStats(delays)
		pair := keys[k]

		stats = append(stats, NasPairStats{
			Request:     pair.req,
			Response:    pair.resp,
			StatsResult: pairStats,
		})
	}
	return stats
}

// getProcedureStats calculates statistics for all tracked procedures
func (dt *DelayTracker) getProcedureStats() map[string]ProcedureStats {
	stats := make(map[string]ProcedureStats)

	for proc, durations := range dt.procHistory {
		if len(durations) == 0 {
			continue
		}

		procStats := calculateStats(durations)

		stats[proc] = ProcedureStats{
			Name:        proc,
			StatsResult: procStats,
			Last:        durations[len(durations)-1],
		}
	}

	return stats
}

func calculateStats(X []float64) StatsResult {
	n := len(X)
	if n == 0 {
		return StatsResult{}
	}

	var min, max, sum float64
	min = X[0]
	max = min
	for _, v := range X {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	mean := sum / float64(n)

	var totalSquaredDiff float64
	for _, v := range X {
		diff := v - mean
		totalSquaredDiff += diff * diff
	}
	variance := totalSquaredDiff / (float64(n) - 1)
	stddev := math.Sqrt(variance)

	return StatsResult{
		Min:    min,
		Max:    max,
		Mean:   mean,
		StdDev: stddev,
		Count:  n,
	}
}
