package scenarios

import (
	"math"
	"strings"

	"stormsim/internal/common/logger"
	"stormsim/internal/common/pool"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/monitoring/oambackend"
	"stormsim/pkg/model"
	"sync"
)

var oamLogger *logger.Logger

func init() {
	oamLogger = logger.InitLogger("info", map[string]string{"mod": "oam"})
}

/********************** remote api **********************/
var oamApi = &RemoteApi{}

type RemoteApi struct {
	UEs  sync.Map // msin string: *uecontext.UeContext
	Gnbs sync.Map // gnbId string: *gnbcontext.GnbContext
}

var ValidMMStates = []string{
	"Deregistered",
	"DeregistrationInitiated",
	"AuthenticationInitiated",
	"RegisteredInitiated",
	"Registered",
}

func matchState(currentState model.StateType, stateArg string) bool {
	stateArg = strings.TrimSpace(stateArg)
	return strings.Contains(string(currentState), stateArg)
}

func (api *RemoteApi) addGnbs(gnbs map[string]*gnbcontext.GnbContext) {
	for gnbId, gnbCtx := range gnbs {
		api.Gnbs.Store(gnbId, gnbCtx)
	}
}
func (api *RemoteApi) addUes(ue *uecontext.UeContext) {
	api.UEs.Store(ue.GetMsin(), ue)
}

func (api *RemoteApi) RemoteGetListUes(level, state, notState string, last int) []string {
	ues := []string{}
	api.UEs.Range(func(key, value any) bool {
		uectx := value.(*uecontext.UeContext)

		if state != "" {
			currentState := uectx.GetMMState().CurrentState()
			if !matchState(currentState, state) {
				return true
			}
		}

		if notState != "" {
			currentState := uectx.GetMMState().CurrentState()
			if matchState(currentState, notState) {
				return true
			}
		}

		if level == "" {
			ues = append(ues, uectx.GetMsin())
		} else {
			level_Logs := uectx.GetLogsByLevel(level)
			if len(level_Logs) > 0 {
				ues = append(ues, uectx.GetMsin())
			}
		}

		return true
	})

	// If last is specified and greater than 0, return only the last N results
	if last > 0 && last < len(ues) {
		ues = ues[len(ues)-last:]
	}

	return ues
}

func (api *RemoteApi) RemoteGetListGnbs() []string {
	allGnbs := []string{}
	api.Gnbs.Range(func(key, value any) bool {
		info := value.(*gnbcontext.GnbContext)
		allGnbs = append(allGnbs, info.GetId())
		return true
	})
	return allGnbs
}

func (api *RemoteApi) RemoteGetUeCtx(msin string) (ueCtx oambackend.UeContextInfo) {
	if value, ok := api.UEs.Load(msin); ok {
		info := value.(*uecontext.UeContext)
		ueCtx = oambackend.UeContextInfo{
			Name: info.GetMsin(),
			Gnb:  info.GetGnbId(),
		}
	}

	return
}
func (api *RemoteApi) RemoteGetSessions(msin string) (info oambackend.SessionInfo) {
	//TODO:
	return
}
func (api *RemoteApi) RemoteMmWorkerStats() (info oambackend.WorkerInfo) {
	info = oambackend.WorkerInfo{
		NumWorkers:        pool.MmWorkerPool.RunningWorkers(),
		NumSubmittedTasks: pool.MmWorkerPool.SubmittedTasks(),
		NumWaitingTasks:   pool.MmWorkerPool.WaitingTasks(),
		NumDroppedTasks:   pool.MmWorkerPool.DroppedTasks(),
		NumCompletedTasks: pool.MmWorkerPool.CompletedTasks(),
	}
	return
}

func (api *RemoteApi) RemoteSmWorkerStats() (info oambackend.WorkerInfo) {
	info = oambackend.WorkerInfo{
		NumWorkers:        pool.SmWorkerPool.RunningWorkers(),
		NumSubmittedTasks: pool.SmWorkerPool.SubmittedTasks(),
		NumWaitingTasks:   pool.SmWorkerPool.WaitingTasks(),
		NumDroppedTasks:   pool.SmWorkerPool.DroppedTasks(),
		NumCompletedTasks: pool.SmWorkerPool.CompletedTasks(),
	}
	return
}

// look for the GnbContext then create its remote API implementation
func (api *RemoteApi) RemoteGetGnbApi(gnbId string) oambackend.GnbApi {
	var gnbCtx *gnbcontext.GnbContext
	if info, ok := api.Gnbs.Load(gnbId); ok {
		gnbCtx = info.(*gnbcontext.GnbContext)
	}

	if gnbCtx == nil {
		return nil
	}
	return &gnbcontext.RemoteGnbApi{
		Ctx: gnbCtx,
	}
}

// look for the UeContext then create its remote API implementation
func (api *RemoteApi) RemoteGetUeApi(ueId string) oambackend.UeApi {
	var ueCtx *uecontext.UeContext
	if info, ok := api.UEs.Load(ueId); ok {
		ueCtx = info.(*uecontext.UeContext)
	}

	if ueCtx == nil {
		return nil
	}
	return &uecontext.RemoteUeApi{
		Ctx: ueCtx,
	}
}

// RemoteGetAllUeDelayStats computes aggregated delay statistics across all UEs
func (api *RemoteApi) RemoteGetAllUeDelayStats() oambackend.GroupDelayStats {
	var totalMean float64
	var totalMeanSq float64 // Sum of squared means
	var totalCount int
	var ueCount int

	// Aggregators for procedures
	procSums := make(map[string]float64)
	procSumSqs := make(map[string]float64) // Sum of squared Last durations
	procCounts := make(map[string]int)
	procMins := make(map[string]float64)
	procMaxs := make(map[string]float64)

	// Aggregators for NAS pairs
	pairSums := make(map[string]float64)
	pairSumSqs := make(map[string]float64) // Sum of squared delays (reconstructed)
	pairCounts := make(map[string]int)
	pairMins := make(map[string]float64)
	pairMaxs := make(map[string]float64)
	pairKeys := make(map[string]struct{ req, resp string })

	api.UEs.Range(func(key, value any) bool {
		ueCtx := value.(*uecontext.UeContext)
		stats := ueCtx.GetDelayStats()
		if stats.Count > 0 {
			totalMean += stats.Mean
			totalMeanSq += stats.Mean * stats.Mean
			totalCount += stats.Count
			ueCount++
		}

		// Aggregate procedure stats
		for name, proc := range stats.Procedures {
			// Use Last duration from each UE as the sample
			procSums[name] += proc.Last
			procSumSqs[name] += proc.Last * proc.Last

			if procCounts[name] == 0 {
				procMins[name] = proc.Last
				procMaxs[name] = proc.Last
			} else {
				if proc.Last < procMins[name] {
					procMins[name] = proc.Last
				}
				if proc.Last > procMaxs[name] {
					procMaxs[name] = proc.Last
				}
			}
			procCounts[name]++
		}

		// Aggregate NAS pair stats
		for _, pair := range stats.NasPairs {
			k := pair.Request + "|" + pair.Response
			// Reconstruct sum from mean * count
			pairSums[k] += pair.Mean * float64(pair.Count)

			// Reconstruct sum of squares from variance + mean^2
			// Variance = StdDev^2
			// SumSq = (Variance * (Count-1)) + (Mean^2 * Count)
			variance := pair.StdDev * pair.StdDev
			sumSq := (variance * float64(pair.Count-1)) + (pair.Mean * pair.Mean * float64(pair.Count))
			if pair.Count == 1 {
				sumSq = pair.Mean * pair.Mean // Special case for count=1 where variance term is 0 (or undefined division)
			}
			pairSumSqs[k] += sumSq

			if pairCounts[k] == 0 {
				pairMins[k] = pair.Min
				pairMaxs[k] = pair.Max
			} else {
				if pair.Min < pairMins[k] {
					pairMins[k] = pair.Min
				}
				if pair.Max > pairMaxs[k] {
					pairMaxs[k] = pair.Max
				}
			}
			pairCounts[k] += pair.Count
			pairKeys[k] = struct{ req, resp string }{pair.Request, pair.Response}
		}

		return true
	})

	if ueCount == 0 {
		return oambackend.GroupDelayStats{}
	}

	// Calculate group mean stddev
	var groupStdDev float64
	if ueCount > 1 {
		groupVariance := (totalMeanSq - (totalMean*totalMean)/float64(ueCount)) / float64(ueCount-1)
		if groupVariance > 0 {
			groupStdDev = math.Sqrt(groupVariance)
		}
	}

	// Build result procedures map
	groupProcs := make(map[string]oambackend.ProcedureStats)
	for name, sum := range procSums {
		count := procCounts[name]
		if count > 0 {
			mean := sum / float64(count)
			var stdDev float64
			if count > 1 {
				variance := (procSumSqs[name] - (sum*sum)/float64(count)) / float64(count-1)
				if variance > 0 {
					stdDev = math.Sqrt(variance)
				}
			}

			groupProcs[name] = oambackend.ProcedureStats{
				Name:   name,
				Mean:   mean,
				StdDev: stdDev,
				Count:  count,
				Min:    procMins[name],
				Max:    procMaxs[name],
			}
		}
	}

	// Build result NAS pairs list
	groupPairs := make([]oambackend.NasPairStats, 0, len(pairKeys))
	for k, key := range pairKeys {
		sum := pairSums[k]
		count := pairCounts[k]
		if count > 0 {
			mean := sum / float64(count)
			var stdDev float64
			if count > 1 {
				variance := (pairSumSqs[k] - (sum*sum)/float64(count)) / float64(count-1)
				if variance > 0 {
					stdDev = math.Sqrt(variance)
				}
			}

			groupPairs = append(groupPairs, oambackend.NasPairStats{
				Request:  key.req,
				Response: key.resp,
				Min:      pairMins[k],
				Max:      pairMaxs[k],
				Mean:     mean,
				StdDev:   stdDev,
				Count:    count,
			})
		}
	}

	return oambackend.GroupDelayStats{
		Mean:       totalMean / float64(ueCount),
		StdDev:     groupStdDev,
		Count:      totalCount,
		UeCount:    ueCount,
		Procedures: groupProcs,
		NasPairs:   groupPairs,
	}
}
