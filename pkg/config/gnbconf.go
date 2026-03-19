package config

import (
	"stormsim/pkg/model"
	"net"
)

type GNodeBConfig struct {
	DefaultControlIF model.ControlIF `yaml:"controlif"`
	DefaultDataIF    model.DataIF    `yaml:"dataif"`
	ListGnbs         []model.GnbInfo `yaml:"listGnbs"`
}

func resolvHost(hostType string, hostOrIp string) string {
	ips, err := net.LookupIP(hostOrIp)
	if err != nil {
		configLogger.Error("Unable to resolve %s in configuration for %s, make sure it is an IP address or a domain that can be resolved to an IPv4", hostOrIp, hostType)
		configLogger.Fatal("DNS lookup failed: %v", err)
	}
	for _, ip := range ips {
		if ip.To4() == nil {
			configLogger.Warn("Skipping %s for host %s as %s, as it is not an IPv4", ip, hostOrIp, hostType)
		} else {
			configLogger.Info("Selecting %s for host %s as %s", ip.String(), hostOrIp, hostType)
			return ip.String()
		}
	}
	configLogger.Fatal("No suitable IP address found as host %s, for %s", hostOrIp, hostType)
	return ""
}
