package scenarios

import (
	"context"
	"fmt"
	"stormsim/internal/common/ds"
	"stormsim/internal/common/fsm"
	"stormsim/internal/core/gnbcontext"
	"stormsim/internal/core/uecontext"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"strconv"
	"sync"
	"time"
)

type UeGroup struct {
	id           int
	nUes         int
	remoteUEs    map[int]*UeRemote
	gnbs         func() (*gnbcontext.GnbContext, bool)
	curGnb       *gnbcontext.GnbContext
	defaultUECfg config.UeConfig
	GroupEvents  chan uecontext.EventUeData // event pool
	Delay        uint16                     // delay between UE
	wg           *sync.WaitGroup
	ctx          context.Context
	mu           sync.RWMutex
}

type UeRemote struct {
	id        int
	msin      string
	fsm_state *fsm.State
	tasks     *ds.Tasks[*uecontext.EventUeData]
}

func newUeGroup(
	id int,
	nUes int,
	gnbs map[string]*gnbcontext.GnbContext,
	defaultUe config.UeConfig,
	logBufferSize int,
	wg *sync.WaitGroup,
	ctx context.Context,
) *UeGroup {
	g := UeGroup{
		id:           id,
		nUes:         nUes,
		remoteUEs:    make(map[int]*UeRemote, nUes),
		gnbs:         getNextGnb(gnbs),
		defaultUECfg: defaultUe,
		GroupEvents:  make(chan uecontext.EventUeData),
		Delay:        defaultUe.Delay,
		wg:           wg,
		ctx:          ctx,
	}
	wg.Add(1)

	g.initUEforGroup(logBufferSize)

	go g.listenEvent(wg)
	time.Sleep(time.Second)
	return &g
}

func (g *UeGroup) listenEvent(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-g.ctx.Done():
			g.close()
			return
		case event, ok := <-g.GroupEvents:
			if !ok {
				testLogger.Error("Cannot assign Event %s to UE", event.EventType)
				continue
			}
			g.mu.RLock()

			g.distributeEvent(&event) // send event to UEs
			g.mu.RUnlock()
		}
	}
}

func (g *UeGroup) close() {
	// terminate all UE
	g.distributeEvent(&uecontext.EventUeData{
		EventType: model.Terminate,
	})
}

func (g *UeGroup) distributeEvent(event *uecontext.EventUeData) {
	if event.EventType == model.XnHandover ||
		event.EventType == model.N2Handover {

		var newGnb *gnbcontext.GnbContext
		var ok bool
		for ueid := range g.remoteUEs {
			if g.remoteUEs[ueid].fsm_state.CurrentState() == model.Registered {
				if g.Delay > 0 {
					time.Sleep(time.Duration(g.Delay) * time.Millisecond)
				}
				g.wg.Go(func() {
					newGnb, ok = g.gnbs()
					if event.EventType == model.XnHandover {
						if !ok {
							testLogger.Error("Cannot find target GNB to setup Handover")
							return
						}
						gnbcontext.TriggerXnHandover(g.curGnb, newGnb, int64(ueid))
					} else {
						if !ok {
							testLogger.Error("Cannot find target GNB to setup Handover")
							return
						}
						gnbcontext.TriggerNgapHandover(g.curGnb, newGnb, int64(ueid))
					}
				})
			} else {
				testLogger.Error(
					"[Setup Event] Event [Handover Event] Fail in UE-%s GroupUE-%d",
					g.remoteUEs[ueid].msin, g.id)
			}
		}
	} else {
		for _, ueRemote := range g.remoteUEs {
			if g.Delay > 0 {
				time.Sleep(time.Duration(g.Delay) * time.Millisecond)
			}
			ueRemote.tasks.AssignTask(event)
		}
	}
}

func (g *UeGroup) initUEforGroup(logBufferSize int) {
	currUECfg := g.defaultUECfg
	currUECfg.UeId = 0

	g.curGnb, _ = g.gnbs()
	for range g.nUes {
		currUECfg.UeId += 1
		currUECfg.Msin = incrementMsin(currUECfg.Msin, 0)

		ueCtx := uecontext.CreateUe(
			currUECfg,
			logBufferSize,
			g.id,
			g.curGnb.GetId(),
			false,
			false,
			g.wg,
			g.ctx,
		)
		oamApi.addUes(ueCtx)
		ueTasks := ueCtx.GetEventQueue()

		g.remoteUEs[currUECfg.UeId] = &UeRemote{
			id:        currUECfg.UeId,
			msin:      currUECfg.Msin,
			tasks:     ueTasks,
			fsm_state: ueCtx.GetMMState(),
		}
	}

	var list_ue string = ""
	for _, ue := range g.remoteUEs {
		list_ue += ue.msin + " "
	}
	testLogger.Info("[GroupUE-%d] List UE: %s", g.id, list_ue)

	g.defaultUECfg = currUECfg
}

// increase <= 0 -> increase by 1
func incrementMsin(currentMSIN string, increase int) string {
	currentValue, _ := strconv.ParseInt(currentMSIN, 10, 64)
	var newValue int64
	if increase == 0 {
		newValue = currentValue + 1
	} else if increase > 0 {
		newValue = currentValue + int64(increase)
	}
	currentMSIN = fmt.Sprintf("%010d", newValue)
	return currentMSIN
}

func getNextGnb(m map[string]*gnbcontext.GnbContext) func() (*gnbcontext.GnbContext, bool) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	index := 0

	return func() (*gnbcontext.GnbContext, bool) {
		if index >= len(keys) {
			index = 0
		}
		key := keys[index]
		index++
		return (m)[key], true
	}
}
