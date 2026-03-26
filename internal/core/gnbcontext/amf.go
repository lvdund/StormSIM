package gnbcontext

import (
	"fmt"
	"stormsim/internal/transport/sctpngap"

	"github.com/lvdund/ngap/aper"
)

// AMF main states in the GNB Context.
const (
	AMF_INACTIVE   string = "AMF_INACTIVE"
	AMF_ACTIVE     string = "AMF_ACTIVE"
	AMF_OVERLOADED string = "AMF_OVERLOAD"
)

type GnbAmfContext struct {
	amfIp              string         // AMF ip
	amfPort            int            // AMF port
	amfId              int64          // AMF id
	tlnaAssoc          TNLAssociation // AMF sctp associations
	amfCapacityValue   int64          // AMF capacity
	state              string
	name               string // amf name.
	regionId           aper.BitString
	setId              aper.BitString
	amfPointer         aper.BitString
	supportedPlmnList  *PlmnSupported
	supportedSliceList *SliceSupported
	lenSlice           int
	lenPlmn            int
	// TODO implement the other fields of the AMF Context
}

// TODO:
type TNLAssociation struct {
	sctpConn     *sctpngap.SctpConn
	weightFactor int64
	usage        aper.Enumerated
	streams      uint16
}

type SliceSupported struct {
	sst    string
	sd     string
	status string
	next   *SliceSupported
}

type PlmnSupported struct {
	mcc  string
	mnc  string
	next *PlmnSupported
}

func (amf *GnbAmfContext) getSliceSupport(index int) (string, string) {

	mov := amf.supportedSliceList
	for range index {
		mov = mov.next
	}

	return mov.sst, mov.sd
}

func (amf *GnbAmfContext) getPlmnSupport(index int) (string, string) {

	mov := amf.supportedPlmnList
	for range index {
		mov = mov.next
	}

	return mov.mcc, mov.mnc
}

func convertMccMnc(plmn string) (mcc string, mnc string) {
	if plmn[2] == 'f' {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c", plmn[5], plmn[4])
	} else {
		mcc = fmt.Sprintf("%c%c%c", plmn[1], plmn[0], plmn[3])
		mnc = fmt.Sprintf("%c%c%c", plmn[2], plmn[5], plmn[4])
	}

	return mcc, mnc
}

func (amf *GnbAmfContext) addedPlmn(plmn string) {

	if amf.lenPlmn == 0 {
		newElem := &PlmnSupported{}

		// newElem.info = plmn
		newElem.next = nil
		newElem.mcc, newElem.mnc = convertMccMnc(plmn)
		// update list
		amf.supportedPlmnList = newElem
		amf.lenPlmn++
		return
	}

	mov := amf.supportedPlmnList
	for range amf.lenPlmn {

		// end of the list
		if mov.next == nil {

			newElem := &PlmnSupported{}
			newElem.mcc, newElem.mnc = convertMccMnc(plmn)
			newElem.next = nil

			mov.next = newElem

		} else {
			mov = mov.next
		}
	}

	amf.lenPlmn++
}

func (amf *GnbAmfContext) addedSlice(sst string, sd string) {

	if amf.lenSlice == 0 {
		newElem := &SliceSupported{}
		newElem.sst = sst
		newElem.sd = sd
		newElem.next = nil

		// update list
		amf.supportedSliceList = newElem
		amf.lenSlice++
		return
	}

	mov := amf.supportedSliceList
	for range amf.lenSlice {

		// end of the list
		if mov.next == nil {

			newElem := &SliceSupported{}
			newElem.sst = sst
			newElem.sd = sd
			newElem.next = nil

			mov.next = newElem

		} else {
			mov = mov.next
		}
	}
	amf.lenSlice++
}

func (amf *GnbAmfContext) setRegionId(regionId []byte) {
	amf.regionId = aper.BitString{
		Bytes:   regionId,
		NumBits: uint64(len(regionId) * 8),
	}
}

func (amf *GnbAmfContext) setSetId(setId []byte) {
	amf.setId = aper.BitString{
		Bytes:   setId,
		NumBits: uint64(len(setId) * 8),
	}
}

func (amf *GnbAmfContext) setPointer(pointer []byte) {
	amf.amfPointer = aper.BitString{
		Bytes:   pointer,
		NumBits: uint64(len(pointer) * 8),
	}
}
