package model

type AMF struct {
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
}
type ControlIF struct {
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
}
type DataIF struct {
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
}

type GnbInfo struct {
	GnbId            string   `yaml:"gnbid"`
	Tac              string   `yaml:"tac"`
	Plmn             Plmn     `yaml:"plmn"`
	SliceSupportList []Snssai `yaml:"slicesupportlist"`
}
