package oambackend

import (
	"strings"

	"github.com/reogac/utils/oam"
)

const (
	STORMSIM_CTX_ID string = "stormsim"
	UE_CTX_PREFIX   string = "ue"
	GNB_CTX_PREFIX  string = "gnb"
)

type Api interface {
	RemoteGetListUes(level, state, notState string, last int) []string // list of ue msin
	RemoteGetListGnbs() []string
	RemoteGetUeCtx(supi string) UeContextInfo
	RemoteGetSessions(supi string) SessionInfo
	RemoteMmWorkerStats() WorkerInfo
	RemoteSmWorkerStats() WorkerInfo
	RemoteGetGnbApi(gnbId string) GnbApi
	RemoteGetUeApi(msin string) UeApi
	RemoteGetAllUeDelayStats() GroupDelayStats
}

func GetOamBackendInfo(api Api) (name, rootId string, getter func(string) *oam.HandlerContext) {
	name = "StormSim"
	rootId = STORMSIM_CTX_ID
	getter = func(contextId string) *oam.HandlerContext {
		if contextId == STORMSIM_CTX_ID { //root context handler (emulator context)
			return oam.NewHandlerContext(contextId, &EmuHandler{
				api: api,
			}, EmuCmds, nil)
		}
		if parts := strings.Split(contextId, ":"); len(parts) == 2 { //ue context handler
			switch parts[0] {
			case UE_CTX_PREFIX: //Ue Context
				if ueApi := api.RemoteGetUeApi(parts[1]); ueApi != nil {
					return oam.NewHandlerContext(contextId, &UeHandler{
						api:    ueApi,
						emuApi: api,
					}, UeCmds, nil)
				}
			case GNB_CTX_PREFIX: //gnb context handler
				if gnbApi := api.RemoteGetGnbApi(parts[1]); gnbApi != nil {
					return oam.NewHandlerContext(contextId, &GnbHandler{
						api:    gnbApi,
						emuApi: api,
					}, GnbCmds, nil)
				}
			}
		}
		return nil
	}
	return
}
