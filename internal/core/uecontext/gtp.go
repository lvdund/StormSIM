package uecontext

import (
	"fmt"

	"stormsim/pkg/model"

	gtpLink "stormsim/monitoring/gtp5g/gogtp5g-link"
	gtpTunnel "stormsim/monitoring/gtp5g/gogtp5g-tunnel"

	"github.com/vishvananda/netlink"

	"net"
	"strconv"
	"strings"
	"time"
)

func (ue *UeContext) setupGtpInterface(msg *model.RlinkSetupPduSessonCommand) {
	gnbPduSession := msg.GetGNBPduSessions()
	pduSession, err := ue.getPduSession(uint8(gnbPduSession.GetPduSessionId()))
	if pduSession == nil || err != nil {
		ue.Error("[GTP] Aborting PDU session %d setup: session not configured on UE side", gnbPduSession.GetPduSessionId())
		return
	}
	select {
	case <-pduSession.ready:
	case <-time.After(1 * time.Second):
		ue.Warn("[GTP] timeout PDU Session [%d] is not ready to setup GTP tunnel!", pduSession.id)
		return
	}

	pduSession.gnbPduSession = gnbPduSession

	if pduSession.id != 1 {
		ue.Warn("[GTP] Only one tunnel per UE is supported for now, no tunnel will be created for second PDU Session of given UE")
		return
	}

	// get UE GNB IP.
	pduSession.ueGnbIP = net.ParseIP(msg.GnbIp)

	upfIp := pduSession.gnbPduSession.GetUpfIp()
	ueIp := pduSession.ueIP
	nameInf := fmt.Sprintf("val%s", ue.msin)
	vrfInf := fmt.Sprintf("vrf%s", ue.msin)
	stopSignal := make(chan bool)

	_ = gtpLink.CmdDel(nameInf)

	if pduSession.stopSignal != nil {
		close(pduSession.stopSignal)
		time.Sleep(time.Second)
	}
	pduSession.stopSignal = stopSignal

	go func() {
		// This function should not return as long as the GTP-U UDP socket is open
		if err := gtpLink.CmdAdd(nameInf, 1, pduSession.ueGnbIP.String(), stopSignal); err != nil {
			ue.Fatal("[GTP] Failed to create kernel GTP interface %s: %v", nameInf, err)
			return
		}
	}()

	cmdAddFar := []string{nameInf, "1", "--action", "2"}
	ue.Debug("[GTP] Configuring FAR: %s", strings.Join(cmdAddFar, " "))
	if err := gtpTunnel.CmdAddFAR(cmdAddFar); err != nil {
		ue.Fatal("[GNB][GTP] Unable to create FAR: %v", err)
		return
	}

	cmdAddFar = []string{nameInf, "2", "--action", "2", "--hdr-creation", "0", fmt.Sprint(gnbPduSession.GetTeidUplink()), upfIp, "2152"}
	ue.Debug("[UE][GTP] Configuring FAR: %s", strings.Join(cmdAddFar, " "))
	if err := gtpTunnel.CmdAddFAR(cmdAddFar); err != nil {
		ue.Fatal("[UE][GTP] Unable to create FAR %v", err)
		return
	}

	cmdAddPdr := []string{nameInf, "1", "--pcd", "1", "--hdr-rm", "0", "--ue-ipv4", ueIp, "--f-teid", fmt.Sprint(gnbPduSession.GetTeidDownlink()), msg.GnbIp, "--far-id", "1"}
	ue.Debug("[GTP] Setting up GTP Packet Detection Rule for %s", strings.Join(cmdAddPdr, " "))

	if err := gtpTunnel.CmdAddPDR(cmdAddPdr); err != nil {
		ue.Fatal("[GNB][GTP] Unable to create FAR: %v", err)
		return
	}

	cmdAddPdr = []string{nameInf, "2", "--pcd", "2", "--ue-ipv4", ueIp, "--far-id", "2"}
	ue.Debug("[GTP] Setting Up GTP Packet Detection Rule for %s", strings.Join(cmdAddPdr, " "))
	if err := gtpTunnel.CmdAddPDR(cmdAddPdr); err != nil {
		ue.Fatal("[UE][GTP] Unable to create FAR %v", err)
		return
	}

	netUeIp := net.ParseIP(ueIp)
	// add an IP address to a link device.
	addrTun := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   netUeIp.To4(),
			Mask: net.IPv4Mask(255, 255, 255, 255),
		},
	}

	link, _ := netlink.LinkByName(nameInf)
	pduSession.tunInterface = link

	if err := netlink.AddrAdd(link, addrTun); err != nil {
		ue.Fatal("[DATA] Error in adding IP for virtual interface %v", err)
		return
	}

	tableId, _ := strconv.Atoi(fmt.Sprint(gnbPduSession.GetTeidUplink()))

	switch ue.tunnelMode {
	case model.TunnelTun:
		rule := netlink.NewRule()
		rule.Priority = 100
		rule.Table = tableId
		rule.Src = addrTun.IPNet
		_ = netlink.RuleDel(rule)

		if err := netlink.RuleAdd(rule); err != nil {
			ue.Fatal("[DATA] Unable to create routing policy rule for UE %v", err)
			return
		}
		pduSession.routingRule = rule
	case model.TunnelVrf:
		vrfDevice := &netlink.Vrf{
			LinkAttrs: netlink.LinkAttrs{
				Name: vrfInf,
			},
			Table: uint32(tableId),
		}
		_ = netlink.LinkDel(vrfDevice)

		ue.Warn("Disable netlink")

		if err := netlink.LinkAdd(vrfDevice); err != nil {
			ue.Fatal("[UE][DATA] Unable to create VRF for UE %v", err)
			return
		}

		if err := netlink.LinkSetMaster(link, vrfDevice); err != nil {
			ue.Fatal("[UE][DATA] Unable to set GTP tunnel as slave of VRF interface %v", err)
			return
		}

		if err := netlink.LinkSetUp(vrfDevice); err != nil {
			ue.Fatal("[UE][DATA] Unable to set interface VRF UP %v", err)
			return
		}
		pduSession.vrf = vrfDevice
	}

	route := &netlink.Route{
		Dst:       &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)}, // default
		LinkIndex: link.Attrs().Index,                                      // dev val<MSIN>
		Scope:     netlink.SCOPE_LINK,                                      // scope link
		Protocol:  4,                                                       // proto static
		Priority:  1,                                                       // metric 1
		Table:     tableId,                                                 // table <ECI>
	}

	if err := netlink.RouteReplace(route); err != nil {
		ue.Fatal("[GTP] Unable to create Kernel Route %v", err)
	}
	pduSession.routeTun = route

	ue.Info("[GTP] Interface %s has successfully been configured for UE %s", nameInf, ueIp)
	switch ue.tunnelMode {
	case model.TunnelTun:
		ue.Info("[GTP] You can do traffic for this UE by binding to IP %s", ueIp)
		ue.Info("[GTP] iperf3 -B %s -c IPERF_SERVER -p PORT -t 9000", ueIp)
	case model.TunnelVrf:
		ue.Info("[GTP] You can do traffic for this UE using VRF %s, eg:", vrfInf)
		ue.Info("[UE][GTP] sudo ip vrf exec %s iperf3 -c IPERF_SERVER -p PORT -t 9000", vrfInf)
	}
}
