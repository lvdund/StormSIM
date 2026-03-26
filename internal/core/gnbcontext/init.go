package gnbcontext

import (
	"context"
	"stormsim/pkg/model"

	"sync"
)

var Gnbs sync.Map // gnbid string: *GnbContext

func InitGnb(
	controlIF model.ControlIF,
	dataIF model.DataIF,
	cfg model.GnbInfo,
	amfs []model.AMF,
	logBufferSize int,
	wg *sync.WaitGroup,
	ctx context.Context,
) *GnbContext {
	gnb := &GnbContext{ctx: ctx}
	gnb.newRanGnbContext(
		cfg.GnbId,
		cfg.Plmn.Mcc,
		cfg.Plmn.Mnc,
		cfg.Tac,
		cfg.SliceSupportList[0].Sst,
		cfg.SliceSupportList[0].Sd,
		controlIF.Ip,
		dataIF.Ip,
		controlIF.Port,
		dataIF.Port,
		logBufferSize)

	// init stats
	//logger.SctpConnStats[gnb.GetId()] = &logger.SctpConnStat{}
	//logger.RLinkConnStats[gnb.GetId()] = &logger.RlinkConnStat{}
	//logger.TaskStats[gnb.GetId()] = []*logger.TaskStat{}

	// start communication with AMF (server SCTP).
	for _, amfConfig := range amfs {
		amf := gnb.newGnbAmf(amfConfig.Ip, amfConfig.Port)
		gnb.Info("Initiate connection with AMF - %s:%d", amfConfig.Ip, amfConfig.Port)
		if err := gnb.initConn(amf); err != nil {
			gnb.Fatal("Error in: %v", err)
		} else {
			gnb.Info("SCTP/NGAP service is running")
		}
		gnb.sendNgSetupRequest(amf)
	}

	// start communication with UE
	go gnb.gnbListen()

	go func() {
		// Block until a Terminate signal is received.
		<-ctx.Done()
		gnb.Terminate()
	}()

	Gnbs.Store(gnb.controlPlaneInfo.gnbId, gnb)
	return gnb
}
