package uecontext

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"stormsim/internal/common/ds"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/logger"
	"stormsim/internal/core/uecontext/sec"
	"stormsim/internal/core/uecontext/timer"
	"stormsim/internal/transport/rlink"
	"stormsim/pkg/config"
	"stormsim/pkg/model"
	"strconv"
	"sync"
	"time"

	"github.com/reogac/nas"

	"github.com/reogac/sbi/models"
	"github.com/vishvananda/netlink"
)

type UeContext struct {
	*logger.BufferedLogger
	id uint16

	state_mm    *fsm.State         // Implement state machine
	timerEngine *timer.TimerEngine //timer

	// Task tracking with timing and state (5gmm & 5gsm) management
	eventQueue  *ds.Tasks[*EventUeData]
	taskTracker *taskTracker

	// for testing
	enableReplay bool
	enableFuzz   bool

	gnbId     string
	rlinkConn *rlink.Connection
	drx       *time.Ticker
	sessions  [16]*PduSession

	secCap *nas.UeSecurityCapability
	supi   string
	msin   string
	snn    string
	suci   nas.MobileIdentity
	guti   *nas.Guti
	nasPdu []byte //registration request/service request for resending in security mode complete

	auth   AuthContext          //on-going authentication context
	secCtx *sec.SecurityContext //current security context

	// TODO: Modify config so you can configure these parameters per PDUSession
	dnn        string
	snssai     models.Snssai
	tunnelMode model.TunnelMode

	mutex sync.Mutex
	wg    *sync.WaitGroup
	ctx   context.Context
}

func CreateUe(
	conf config.UeConfig,
	logBufferSize int,
	groupId int,
	gnbId string,
	enableReplay bool,
	enableFuzz bool,
	wg *sync.WaitGroup,
	ctx context.Context,
) *UeContext {
	sst, _ := strconv.Atoi(conf.Snssai.Sst)
	ue := &UeContext{
		id:     uint16(conf.UeId),
		msin:   conf.Msin,
		secCap: conf.GetUESecurityCapability(),
		snssai: models.Snssai{Sd: conf.Snssai.Sd, Sst: sst},
		ctx:    ctx,
		wg:     wg,
	}

	ue.BufferedLogger = logger.NewBufferedLogger(
		logBufferSize,
		"ue",
		conf.Msin,
		map[string]string{"mod": "ue", "msin": conf.Msin, "group": strconv.Itoa(groupId)},
		func() string {
			if ue.state_mm != nil {
				return string(ue.state_mm.CurrentState())
			}
			return ""
		},
	)

	// init AuthContext
	key, _ := hex.DecodeString(conf.Key)
	if len(conf.Opc) > 0 {
		op, _ := hex.DecodeString(conf.Opc)
		ue.auth.milenage, _ = sec.NewMilenage(key, op, true) //use OPC
	} else {
		op, _ := hex.DecodeString(conf.Op)
		ue.auth.milenage, _ = sec.NewMilenage(key, op, false) //use OP
	}
	ue.auth.amf, _ = hex.DecodeString(conf.Amf)
	sqn, _ := hex.DecodeString(conf.Sqn)
	ue.auth.sqn.Set(sqn)

	// add supi
	mcc := conf.Hplmn.Mcc
	mnc := conf.Hplmn.Mnc
	ue.auth.supi = fmt.Sprintf("imsi-%s%s%s", mcc, mnc, conf.Msin)
	ue.createSuci(mcc, mnc)
	// ue.createConcealSuci(mcc, mnc, conf) ///////////////TODO: need check profile A, B

	// add Data Network Name.
	ue.dnn = conf.Dnn
	ue.tunnelMode = conf.TunnelMode

	// init state machine: 5gmm
	ue.state_mm = fsm.NewState(model.Deregistered, ue)
	ue.timerEngine = timer.NewTimerEngine()

	ue.connectGnb(gnbId) // listening gnb signal
	ue.newTasks()        // listening remote event/task

	// testing
	if enableFuzz {
		ue.enableFuzz = enableFuzz
		now := time.Now()
		tail := now.Format("_150405") // HHMMSS
		session_id := fmt.Sprintf("session%s", tail)
		Capture = NewMessageCapture(session_id)
		Capture.AddMetadata("description", "Test capture session")
		Capture.AddMetadata("version", "1.0")
	}
	if enableReplay {
		ue.enableReplay = enableReplay
		ue.Info("Session ID: %s\n", Replay.session.SessionID)
		ue.Info("Message count: %d\n", len(Replay.session.Messages))
		wg.Go(func() {
			ue.triggerReplay()
		})
	}

	return ue
}

func (ue *UeContext) GetMsin() string {
	return ue.msin
}
func (ue *UeContext) GetGnbId() string {
	return ue.gnbId
}

// GetId returns the UE's internal ID for handover operations
func (ue *UeContext) GetId() int64 {
	return int64(ue.id)
}

func (ue *UeContext) GetEventQueue() *ds.Tasks[*EventUeData] {
	return ue.eventQueue
}
func (ue *UeContext) GetMMState() *fsm.State {
	return ue.state_mm
}

func (ue *UeContext) resetSecurityContext() {
	ue.secCtx = nil
	ue.auth.ngKsi.Id = 7
}

func (ue *UeContext) createPDUSession() (*PduSession, error) {
	pduSession := &PduSession{}

	// pdu session index
	pduSessionIndex := -1
	for i, pduSession := range ue.sessions {
		if pduSession == nil && i > 0 { //session id zero is reserved (3GPP)
			pduSessionIndex = i
			break
		}
	}
	if pduSessionIndex == -1 {
		return nil, errors.New("unable to create an additional PDU Session, we already created the max number of PDU Session")
	}
	pduSession.id = uint8(pduSessionIndex)

	// add logger
	pduSession.Logger = logger.InitLogger("", map[string]string{
		"mod":     "pdu",
		"ue":      ue.msin,
		"session": strconv.Itoa(pduSessionIndex),
	})

	// create 5gsm state machine
	pduSession.state_sm = fsm.NewState(model.PDUSessionInactive, pduSession)
	pduSession.wait = make(chan bool)
	pduSession.ready = make(chan bool)

	pduSession.ueCtx = ue
	ue.sessions[pduSessionIndex] = pduSession

	return pduSession, nil
}

// return current nas security context for encoding/decoding nas message
func (ue *UeContext) getNasContext() *nas.NasContext {
	if ue.secCtx != nil {
		return ue.secCtx.NasContext(true)
	}
	return nil
}

func (ue *UeContext) sendEventMm(event *fsm.EventData) chan error {
	return _uePool.fsm_mm.SendEvent(ue.state_mm, event)
}

func (ue *UeContext) getDRX() <-chan time.Time {
	if ue.drx == nil {
		return nil
	}
	return ue.drx.C
}

func (ue *UeContext) stopDRX() {
	if ue.drx != nil {
		ue.drx.Stop()
	}
}

func (ue *UeContext) createDRX(d time.Duration) {
	ue.drx = time.NewTicker(d)
}

func (ue *UeContext) getPduSession(pduSessionId uint8) (*PduSession, error) {
	if pduSessionId < 1 || pduSessionId >= 16 || ue.sessions[pduSessionId] == nil {
		return nil, errors.New("Unable to find GnbPDUSession ID " + string(pduSessionId))
	}
	return ue.sessions[pduSessionId], nil
}

func (ue *UeContext) deletePduSession(pduSessionId uint8) error {
	if pduSessionId < 1 || pduSessionId >= 16 || ue.sessions[pduSessionId] == nil {
		return errors.New("Unable to find GnbPDUSession ID " + string(pduSessionId))
	}
	pduSession := ue.sessions[pduSessionId]
	close(pduSession.wait)
	if _, ok := <-pduSession.ready; ok {
		close(pduSession.ready)
	}
	if pduSession.stopSignal != nil {
		pduSession.stopSignal <- true
	}
	pduSession.Info("Successfully released PDU Session")
	ue.sessions[pduSessionId] = nil
	return nil
}

// NOTE: for open5gs
func (ue *UeContext) createConcealSuci(mcc, mnc string, ueConf config.UeConfig) {
	var plmnId nas.PlmnId
	var route nas.RoutingIndicator

	plmnId.Set(mcc, mnc)
	route.Parse(ueConf.RoutingIndicator)

	suci := new(nas.SupiImsi)
	suci.Parse([]string{mcc, mnc, ueConf.RoutingIndicator, ueConf.ProtectionScheme, ueConf.HomeNetworkPublicKeyID, ue.msin})

	ue.suci = nas.MobileIdentity{Id: &nas.Suci{Content: suci}}
}

func (ue *UeContext) createSuci(mcc, mnc string) {
	var plmnId nas.PlmnId
	plmnId.Set(mcc, mnc)

	suci := new(nas.SupiImsi)
	suci.Parse([]string{plmnId.String(), ue.msin})
	ue.suci = nas.MobileIdentity{
		Id: &nas.Suci{
			Content: suci,
		},
	}
}

func (ue *UeContext) get5GTmsi() nas.MobileIdentity {
	amfId := ue.guti.AmfId
	stmsi := &nas.Tmsi5Gs{
		Tmsi: ue.guti.Tmsi,
	}
	stmsi.AmfId.Set(0, amfId.GetSet(), amfId.GetPointer())
	return nas.MobileIdentity{
		Id: stmsi,
	}
}

func (ue *UeContext) getTmsiBytes() (tmsi [4]uint8) {
	if id := ue.guti; id != nil {
		binary.BigEndian.PutUint32(tmsi[:], id.Tmsi)
	}
	return
}
func (ue *UeContext) set5gGuti(guti *nas.MobileIdentity) {
	if guti.GetType() != nas.MobileIdentity5GSType5gGuti {
		//TODO: warn
		return
	}
	ue.guti = guti.Id.(*nas.Guti)
}

func (ue *UeContext) Terminate() {
	// clean all context of tun interface
	for _, pduSession := range ue.sessions {
		if pduSession != nil {
			ueTunInf := pduSession.tunInterface
			ueRoutingRule := pduSession.routingRule
			ueRouteTun := pduSession.routeTun
			ueVrf := pduSession.vrf

			if ueTunInf != nil {
				_ = netlink.LinkSetDown(ueTunInf)
				_ = netlink.LinkDel(ueTunInf)
			}

			if ueRoutingRule != nil {
				_ = netlink.RuleDel(ueRoutingRule)
			}

			if ueRouteTun != nil {
				_ = netlink.RouteDel(ueRouteTun)
			}

			if ueVrf != nil {
				_ = netlink.LinkSetDown(ueVrf)
				_ = netlink.LinkDel(ueVrf)
			}
		}
	}

	ue.mutex.Lock()
	ue.rlinkConn.Close()
	if ue.drx != nil {
		ue.drx.Stop()
	}
	ue.mutex.Unlock()

	ue.Info("UE context terminated successfully")
}
