package uecontext

import (
	"fmt"
	"net"
	"stormsim/internal/common/fsm"
	"stormsim/internal/common/logger"
	"stormsim/pkg/model"

	"github.com/vishvananda/netlink"
)

// 5GSM main states in the UE.
const (
	SM5G_PDU_SESSION_INACTIVE uint8 = iota
	SM5G_PDU_SESSION_ACTIVE_PENDING
	SM5G_PDU_SESSION_ACTIVE
)

type PduSession struct {
	*logger.Logger
	id              uint8
	gnbPduSession   *model.GnbPDUSessionContext
	ueIP            string
	ueGnbIP         net.IP
	tunInterface    netlink.Link
	routingRule     *netlink.Rule
	routeTun        *netlink.Route
	vrf             *netlink.Vrf
	stopSignal      chan bool // stop GTP tunnel interface
	wait            chan bool // Synchronization for PDU session state transitions
	ready           chan bool // for begin setup GTP tunnel
	t3580RetrtCount int

	// TS 24.501 - 6.1.3.2.1.1 State Machine for Session Management
	ueCtx    *UeContext
	state_sm *fsm.State
}

func (pduSession *PduSession) SendEventSm(event *fsm.EventData) chan error {
	return _uePool.fsm_sm.SendEvent(pduSession.state_sm, event)
}
func (pduSession *PduSession) SendSyncEventSm(event *fsm.EventData) error {
	return _uePool.fsm_sm.SyncSendEvent(pduSession.state_sm, event)
}

func (pduSession *PduSession) setIp(ip []uint8) {
	pduSession.ueIP = fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
	// pduSession.ready <- true
	// close(pduSession.ready)
}
