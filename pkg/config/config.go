package config

import (
	"os"
	"stormsim/internal/common/logger"
	"stormsim/pkg/model"

	"github.com/reogac/nas"
	"gopkg.in/yaml.v2"
)

var configLogger *logger.Logger
var configPathString string = "./config/config.yml"

func init() {
	configLogger = logger.InitLogger("info", map[string]string{"mod": "config"})
}

type LoggingConfig struct {
	UeLogBufferSize  int `yaml:"ueLogBufferSize"`
	GnbLogBufferSize int `yaml:"gnbLogBufferSize"`
}

type Config struct {
	GNodeBConfig GNodeBConfig  `yaml:"gnodeb"`
	DefaultUe    UeConfig      `yaml:"defaultUe"`
	AMFs         []model.AMF   `yaml:"amfif"`
	Scenarios    []Scenario    `yaml:"scenarios"`
	RemoteServer RemoteServer  `yaml:"remote"`
	Testing      TestingConf   `yaml:"testconf"`
	LogLevel     string        `yaml:"loglevel"`
	Logging      LoggingConfig `yaml:"logging"`
}

type RemoteServer struct {
	Enable bool   `yaml:"enable"`
	Ip     string `yaml:"ip"`
	Port   int    `yaml:"port"`
}

// `Scenario` is a group of ues
// each group have its own setup behaviour list of `EventInfo`
type Scenario struct {
	NUEs     int         `yaml:"nUEs"`
	Gnbs     []string    `yaml:"gnbs"`
	UeEvents []EventInfo `yaml:"ueEvents,omitempty"`
}

type EventInfo struct {
	Event                 model.EventType `yaml:"event"`
	TimeBeforeExcuteEvent uint8           `yaml:"delay,omitempty"`

	// event info
	NumberPduSessions int `yaml:"number_pdu_sessions,omitempty"` // max pdus = 16
	RegisterType      int `yaml:"register_type,omitempty"`       // 0: Initial; 1: Emergency
	DeregisterType    int `yaml:"deregister_type,omitempty"`     // 0: not switch off; 1: switch off
	PduSessionType    int `yaml:"pdu_session_type,omitempty"`    // 0: Initial; 1: Emergency

	// include params
	Params []int `yaml:"params,omitempty"`
}

type TestingConf struct {
	EnableFuzz   bool    `yaml:"enableFuzz"`
	EnableReplay bool    `yaml:"enableReplay,omitempty"`
	Mm           RanFuzz `yaml:"5gmm,omitempty"`
	Sm           RanFuzz `yaml:"5gsm,omitempty"`
}

type RanFuzz struct {
	States []model.StateType `yaml:"states,omitempty"`
	Events []model.EventType `yaml:"events,omitempty"`
}

func LoadConfig(configPath string) Config {
	cfg := Config{}
	f, err := os.Open(configPath)
	if err != nil {
		configLogger.Fatal("Could not open config at \"%s\": %v", configPath, err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	if err = decoder.Decode(&cfg); err != nil {
		configLogger.Fatal("Could not unmarshal yaml config at \"%s\": %v", configPath, err)
	}

	for i, amf := range cfg.AMFs {
		cfg.AMFs[i].Ip = resolvHost("AMF's IP address", amf.Ip)
	}

	cfg.GNodeBConfig.DefaultDataIF.Ip = resolvHost("Default gNodeB's N3/Data IP address", cfg.GNodeBConfig.DefaultDataIF.Ip)
	cfg.GNodeBConfig.DefaultControlIF.Ip = resolvHost("Default gNodeB's N2/Control IP address", cfg.GNodeBConfig.DefaultControlIF.Ip)

	logger.ParseLogLevel(cfg.LogLevel)

	if cfg.Logging.UeLogBufferSize == 0 {
		cfg.Logging.UeLogBufferSize = 50
	}
	if cfg.Logging.GnbLogBufferSize == 0 {
		cfg.Logging.GnbLogBufferSize = 100
	}

	configPathString = configPath

	return cfg
}

func GetConfigOrDefaultConf() Config {
	return LoadConfig(configPathString)
}

func (ue *UeConfig) GetUESecurityCapability() *nas.UeSecurityCapability {
	secCap := new(nas.UeSecurityCapability) //2 bytes

	// Ciphering algorithms
	secCap.SetEA(0, ue.Ciphering.Nea0)
	secCap.SetEA(1, ue.Ciphering.Nea1)
	secCap.SetEA(2, ue.Ciphering.Nea2)
	secCap.SetEA(3, ue.Ciphering.Nea3)

	// Integrity algorithms
	secCap.SetIA(0, ue.Integrity.Nia0)
	secCap.SetIA(1, ue.Integrity.Nia1)
	secCap.SetIA(2, ue.Integrity.Nia2)
	secCap.SetIA(3, ue.Integrity.Nia3)
	return secCap
}
