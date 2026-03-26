package gnbcontext

import (
	"fmt"
	"stormsim/internal/common/logger"
	"stormsim/monitoring/oambackend"
	"stormsim/pkg/model"
)

/********************** remote gnb api **********************/
type RemoteGnbApi struct {
	Ctx *GnbContext
}

func (api *RemoteGnbApi) RemoteGnbLogs(last int, level string) []logger.LogEntry {
	if level != "" {
		return api.Ctx.GetLogsByLevel(level)
	}
	if last > 0 {
		return api.Ctx.GetLastLogs(last)
	}
	return api.Ctx.GetLogs()
}

func (api *RemoteGnbApi) RemoteGnbInfo() oambackend.GnbInfo {
	return oambackend.GnbInfo{
		Name:     api.Ctx.controlPlaneInfo.gnbId,
		Plmn:     fmt.Sprintf("%s/%s", api.Ctx.controlPlaneInfo.mcc, api.Ctx.controlPlaneInfo.mnc),
		Snssai:   fmt.Sprintf("%s/%s", api.Ctx.sliceConfiguration.sst, api.Ctx.sliceConfiguration.sd),
		NgapAddr: fmt.Sprintf("%s:%d", api.Ctx.controlPlaneInfo.gnbIp, api.Ctx.controlPlaneInfo.gnbPort),
		GtpAddr:  fmt.Sprintf("%s:%d", api.Ctx.dataPlaneInfo.gnbIp, api.Ctx.dataPlaneInfo.gnbPort),
	}
}

func (api *RemoteGnbApi) RemoteCountUes() int {
	count := 0
	api.Ctx.prUeIdPool.Range(func(key, value any) bool {
		if _, ok := value.(*GnbUeContext); ok {
			count++
		}
		return true
	})
	return count
}

func (api *RemoteGnbApi) RemoteListUeCtxs() []string {
	listue := []string{}
	api.Ctx.prUeIdPool.Range(func(key, value any) bool {
		if ue, ok := value.(*GnbUeContext); ok {
			listue = append(listue, ue.msin)
		}
		return true
	})

	return listue
}

func (api *RemoteGnbApi) RemoteReleaseUe(msin string) bool {
	err := TriggerReleaseUe(api.Ctx, msin)
	if err != nil {
		return false
	}
	return true
}

func (api *RemoteGnbApi) RemoteListAmf() []oambackend.AmfInfo {
	listamf := []oambackend.AmfInfo{}
	api.Ctx.amfPool.Range(func(key, value any) bool {
		if amf, ok := value.(*GnbAmfContext); ok {
			plmnlist := []string{}
			slicelist := []string{}
			for i := range amf.lenPlmn {
				mcc, mnc := amf.getPlmnSupport(i)
				plmnlist = append(plmnlist, fmt.Sprintf("%s/%s", mcc, mnc))
			}
			for i := range amf.lenSlice {
				sst, sd := amf.getSliceSupport(i)
				slicelist = append(slicelist, fmt.Sprintf("%s/%s", sst, sd))
			}
			listamf = append(listamf, oambackend.AmfInfo{
				Name:         amf.name,
				Address:      model.AMF{Ip: amf.amfIp, Port: amf.amfPort},
				State:        amf.state,
				PlmnSupport:  plmnlist,
				SliceSupport: slicelist,
			})
		}
		return true
	})
	return nil
}

// RemoteGnbDelayLogs returns NGAP delay log entries for this gNB
func (api *RemoteGnbApi) RemoteGnbDelayLogs(last int) []oambackend.NasDelayEntry {
	entries := api.Ctx.GetDelayLogs(last)
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

// RemoteGnbDelayStats returns aggregated delay statistics for this gNB
func (api *RemoteGnbApi) RemoteGnbDelayStats() oambackend.DelayStats {
	stats := api.Ctx.GetDelayStats()
	return oambackend.DelayStats{
		Min:    stats.Min,
		Max:    stats.Max,
		Mean:   stats.Mean,
		Count:  stats.Count,
		StdDev: stats.StdDev,
	}
}
