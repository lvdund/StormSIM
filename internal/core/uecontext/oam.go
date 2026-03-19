package uecontext

import (
	"stormsim/internal/common/logger"
	"stormsim/monitoring/oambackend"
	"stormsim/pkg/model"

	"github.com/reogac/sbi/models"
)

type RemoteUeApi struct {
	Ctx *UeContext
}

func (api *RemoteUeApi) RemoteUeInfo() oambackend.UeContextInfo {
	countSS := 0
	for _, ss := range api.Ctx.sessions {
		if ss != nil {
			countSS++
		}
	}
	return oambackend.UeContextInfo{
		Name:           api.Ctx.msin,
		Gnb:            api.Ctx.gnbId,
		Hplmn:          api.Ctx.guti.PlmnId.String(),
		Snssai:         api.Ctx.snssai.String(),
		MMstate:        string(api.Ctx.state_mm.CurrentState()),
		ActiveSessions: int8(countSS),
	}
}

func (api *RemoteUeApi) RemoteUeStats() []string {
	stats := []string{}
	return stats
}

func (api *RemoteUeApi) RemoteUeSessionInfo() []oambackend.SessionInfo {
	sslist := []oambackend.SessionInfo{}
	for _, ss := range api.Ctx.sessions {
		if ss != nil {
			sslist = append(sslist, oambackend.SessionInfo{
				Id:      ss.id,
				Dnn:     api.Ctx.dnn,
				Snssai:  api.Ctx.snssai.String(),
				SMstate: string(ss.state_sm.CurrentState()),
				Address: ss.ueIP,
			})
		}
	}
	return sslist
}

func (api *RemoteUeApi) RemoteCreateSession(dnn string, slice models.Snssai) bool {
	api.Ctx.eventQueue.AssignTask(&EventUeData{
		EventType: model.PduSessionInit,
		Params: map[string]any{
			"dnn":    dnn,
			"snssai": slice,
		},
	})
	return true
}

func (api *RemoteUeApi) RemoteUeLogs(last int, level string) []logger.LogEntry {
	if level != "" {
		return api.Ctx.GetLogsByLevel(level)
	}
	if last > 0 {
		return api.Ctx.GetLastLogs(last)
	}
	return api.Ctx.GetLogs()
}

// RemoteUeDelayLogs returns NAS delay log entries for this UE
func (api *RemoteUeApi) RemoteUeDelayLogs(last int, protocol string) []oambackend.NasDelayEntry {
	entries := api.Ctx.GetDelayLogsByProtocol(protocol, last)
	if entries == nil {
		return nil
	}

	result := make([]oambackend.NasDelayEntry, len(entries))
	for i, e := range entries {
		result[i] = oambackend.NasDelayEntry{
			Protocol:     e.Protocol,
			RequestType:  e.RequestType,
			ResponseType: e.ResponseType,
			SendTime:     e.SendTime.Format("15:04:05.000"),
			DelayMs:      e.DelayMs,
		}
	}
	return result
}

// RemoteUeDelayStats returns aggregated delay statistics for this UE
func (api *RemoteUeApi) RemoteUeDelayStats() oambackend.DelayStats {
	stats := api.Ctx.GetDelayStats()

	procedures := make(map[string]oambackend.ProcedureStats)
	for k, v := range stats.Procedures {
		procedures[k] = oambackend.ProcedureStats{
			Name:  v.Name,
			Min:   v.Min,
			Max:   v.Max,
			Mean:  v.Mean,
			Count: v.Count,
			Last:  v.Last,
		}
	}

	nasPairs := make([]oambackend.NasPairStats, len(stats.NasPairs))
	for i, v := range stats.NasPairs {
		nasPairs[i] = oambackend.NasPairStats{
			Request:  v.Request,
			Response: v.Response,
			Min:      v.Min,
			Max:      v.Max,
			Mean:     v.Mean,
			Count:    v.Count,
		}
	}

	return oambackend.DelayStats{
		Min:        stats.Min,
		Max:        stats.Max,
		Mean:       stats.Mean,
		Count:      stats.Count,
		Procedures: procedures,
		NasPairs:   nasPairs,
	}
}
