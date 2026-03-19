package model

import (
	"stormsim/internal/transport/rlink"

	"github.com/reogac/nas"
)

const (
	RLinkCreateConnectionRequestType      string = "RLinkCreateConnectionRequest message"
	RLinkCreateConnectionResponseType     string = "RLinkCreateConnectionResponse message"
	RlinkUeIdleInformType                 string = "RlinkUeIdleInform message"
	RlinkUeReadyPagingType                string = "RlinkUeReadyPaging message"
	RlinkSetupPagingType                  string = "RlinkSetupPaging message"
	RLinkHandoverPrepareRequestType       string = "RlinkHandoverPrepareRequest message"
	RlinkRlinkHandoverPrepareResponseType string = "RlinkRlinkHandoverPrepareResponse message"
	RLinkHandoverForwardUeContextType     string = "RLinkHandoverForwardUeContext message"

	NasMsgType                     string = "NasMsg message"
	RlinkSetupPduSessonCommandType string = "RlinkSetupPduSessonCommand message"
)

type RlinkSetupPduSessonCommand struct {
	PrUeId         int64
	GNBPduSessions [16]*GnbPDUSessionContext
	GnbIp          string
}

func (sr *RlinkSetupPduSessonCommand) GetType() string {
	return RlinkSetupPduSessonCommandType
}

func (sr *RlinkSetupPduSessonCommand) GetGNBPduSessions() *GnbPDUSessionContext {
	for _, s := range sr.GNBPduSessions {
		if s != nil {
			return s
		}
	}
	return nil
}

type NasMsg struct {
	PrUeId int64
	Nas    []byte
}

func (nm *NasMsg) GetType() string {
	return NasMsgType
}

type RLinkCreateConnectionRequest struct { // to inbound
	PrUeId int64
	Msin   string
	Tmsi   *nas.Guti
	Conn   *rlink.Connection // r t
}

func (in *RLinkCreateConnectionRequest) GetType() string {
	return RLinkCreateConnectionRequestType
}

type RLinkCreateConnectionResponse struct {
	PrUeId int64
	Mcc    string
	Mnc    string
	Conn   *rlink.Connection // r t
}

func (in *RLinkCreateConnectionResponse) GetType() string {
	return RLinkCreateConnectionResponseType
}

type RlinkUeIdleInform struct {
	PrUeId int64
}

func (in *RlinkUeIdleInform) GetType() string {
	return RlinkUeIdleInformType
}

type RlinkUeReadyPaging struct { // to inbound
	PrUeId        int64
	Conn          *rlink.Connection // t
	FetchPagedUEs bool
}

func (pa *RlinkUeReadyPaging) GetType() string {
	return RlinkUeReadyPagingType
}

type RlinkSetupPaging struct {
	PrUeId   int64
	PagedUEs []PagedUE
}

func (pa *RlinkSetupPaging) GetType() string {
	return RlinkSetupPagingType
}

type RLinkHandoverPrepareRequest struct {
	PrUeId       int64
	Conn         *rlink.Connection // r t i : newgnb
	TargetGnbId  string
	IsXnHandover bool
	IsN2Handover bool
}

func (xn *RLinkHandoverPrepareRequest) GetType() string {
	return RLinkHandoverPrepareRequestType
}

type RlinkRlinkHandoverPrepareResponse struct {
	PrUeId           int64
	ConnectionClosed bool
	IsXnHandover     bool
	IsN2Handover     bool
}

func (xn *RlinkRlinkHandoverPrepareResponse) GetType() string {
	return RlinkRlinkHandoverPrepareResponseType
}

type RLinkHandoverForwardUeContext struct { // S-gnb -> T-gnb
	PrUeId int64
	Msin   string
	Conn   *rlink.Connection // r t

	// gnbue info
	AmfUeNgapId   int64
	UeCoreContext *UeCoreContext

	SourceGnbId string
}

func (xn *RLinkHandoverForwardUeContext) GetType() string {
	return RLinkHandoverForwardUeContextType
}
