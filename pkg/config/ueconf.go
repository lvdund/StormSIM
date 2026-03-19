package config

import "stormsim/pkg/model"

type UeConfig struct {
	UeId int    `yaml:"-"`
	Msin string `yaml:"msin"`
	Key  string `yaml:"key"`
	Opc  string `yaml:"opc"`
	Op   string `yaml:"op"`
	Amf  string `yaml:"amf"`
	Sqn  string `yaml:"sqn"`
	Dnn  string `yaml:"dnn"`

	ProtectionScheme       string `yaml:"protectionScheme"`
	HomeNetworkPublicKey   string `yaml:"homeNetworkPublicKey"`
	HomeNetworkPublicKeyID string `yaml:"homeNetworkPublicKeyID"`
	RoutingIndicator       string `yaml:"routingindicator"`

	Hplmn      model.Plmn       `yaml:"hplmn"`
	Snssai     model.Snssai     `yaml:"snssai"`
	Integrity  model.Integrity  `yaml:"integrity"`
	Ciphering  model.Ciphering  `yaml:"ciphering"`
	TunnelMode model.TunnelMode `yaml:"-"`

	Delay uint16 `yaml:"delay"`
}
