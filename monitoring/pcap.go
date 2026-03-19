package monitoring

import (
	"net"
	"os"

	"stormsim/internal/common/logger"
	"stormsim/pkg/config"

	"github.com/urfave/cli/v2"
	"github.com/vishvananda/netlink"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

var log *logger.Logger

func init() {
	log = logger.InitLogger("info", map[string]string{"mod": "pcap"})
}

func CaptureTraffic(path cli.Path) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal("Failed to create pcap file: %v", err)
	}

	config := config.GetConfigOrDefaultConf()
	ip := net.ParseIP(config.GNodeBConfig.DefaultControlIF.Ip)

	links, err := netlink.LinkList()
	if err != nil {
		log.Fatal("Unable to capture traffic to AMF as we are unable to get Links informations: %v", err)
	}
	var n2Link *netlink.Link
outer:
	for _, link := range links {
		addrs, err := netlink.AddrList(link, 0)
		if err != nil {
			log.Error("Unable to get IPs of link %s: %v", link.Attrs().Name, err)
			continue
		}
		for _, addr := range addrs {
			if addr.IP.Equal(ip) {
				n2Link = &link
				break outer
			}
		}
	}

	if n2Link == nil {
		log.Fatal("Unable to find network interface providing gNodeB IP %s", ip)
	}

	pcapw := pcapgo.NewWriter(f)
	if err := pcapw.WriteFileHeader(1600, layers.LinkTypeEthernet); err != nil {
		log.Fatal("WriteFileHeader failed: %v", err)
	}

	handle, err := pcapgo.NewEthernetHandle((*n2Link).Attrs().Name)
	if err != nil {
		log.Fatal("OpenEthernet failed: %v", err)
	}

	pkgsrc := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	go func() {
		for packet := range pkgsrc.Packets() {
			if err := pcapw.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
				log.Fatal("pcap.WritePacket failed: %v", err)
			}
		}
	}()
}
