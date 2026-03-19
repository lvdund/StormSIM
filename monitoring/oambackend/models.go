package oambackend

import "stormsim/pkg/model"

type GnbUeCtxInfo struct {
	Name string
}

type UeContextShortInfo struct {
	Name string // msin
}

type UeContextInfo struct {
	Name           string // msin
	Gnb            string
	Hplmn          string
	Snssai         string
	RanNgapId      int64
	AmfNgapId      int64
	MMstate        string
	ActiveSessions int8
}

type GnbInfo struct {
	Name     string // gnbid
	Plmn     string // mcc/mnc
	Snssai   string // sst/sd
	NgapAddr string
	GtpAddr  string
}
type AmfInfo struct {
	Name         string // amf id
	Address      model.AMF
	State        string
	PlmnSupport  []string // mcc/mnc
	SliceSupport []string // sst/sd
}

type SessionInfo struct {
	Id      uint8
	Dnn     string
	Snssai  string
	SMstate string
	Address string
}

type WorkerInfo struct {
	NumWorkers        int64
	NumSubmittedTasks uint64
	NumWaitingTasks   uint64
	NumDroppedTasks   uint64
	NumCompletedTasks uint64
}

// NasDelayEntry represents a delay measurement for OAM display
type NasDelayEntry struct {
	Protocol     string  `json:"protocol"`
	RequestType  string  `json:"requestType"`
	ResponseType string  `json:"responseName"`
	SendTime     string  `json:"sendTime"` // Formatted time string
	DelayMs      float64 `json:"delayMs"`
}

// ProcedureStats contains statistics for a specific 5G procedure
type ProcedureStats struct {
	Name   string  `json:"name"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stdDev"`
	Count  int     `json:"count"`
	Last   float64 `json:"last"`
}

// NasPairStats contains aggregated statistics for NAS message pairs
type NasPairStats struct {
	Request  string  `json:"request"`
	Response string  `json:"response"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Mean     float64 `json:"mean"`
	StdDev   float64 `json:"stdDev"`
	Count    int     `json:"count"`
}

// DelayStats contains aggregated delay statistics for OAM display
type DelayStats struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
	StdDev float64 `json:"stdDev"`
	Count  int     `json:"count"`

	// Procedures contains statistics for specific 5G procedures
	Procedures map[string]ProcedureStats `json:"procedures"`

	// NasPairs contains statistics for specific NAS message pairs
	NasPairs []NasPairStats `json:"nasPairs"`
}

// GroupDelayStats contains group-level delay statistics for OAM display
type GroupDelayStats struct {
	Mean    float64 `json:"mean"`
	StdDev  float64 `json:"stdDev"`
	Count   int     `json:"count"`
	UeCount int     `json:"ueCount"`

	// Procedures contains aggregated procedure statistics
	Procedures map[string]ProcedureStats `json:"procedures"`

	// NasPairs contains aggregated statistics for NAS message pairs
	NasPairs []NasPairStats `json:"nasPairs"`
}
