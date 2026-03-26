package model

type Plmn struct {
	Mcc string `yaml:"mcc"`
	Mnc string `yaml:"mnc"`
}
type Snssai struct {
	Sst string `yaml:"sst"`
	Sd  string `yaml:"sd"`
}
