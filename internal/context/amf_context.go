package context

import (
	"sync"

	"github.com/gavin/amf/internal/logger"
)

type AMFContext struct {
	Name            string
	GuamiList       []Guami
	ServedGuami     []Guami
	PlmnSupportList []PlmnSupport

	RelativeCapacity int32

	SupportedFeatures map[string]bool

	UeContexts sync.Map

	RanContexts sync.Map

	NfId string

	mu sync.RWMutex
}

type Guami struct {
	PlmnId      PlmnId
	AmfId       string
	AmfRegionId string
	AmfSetId    string
	AmfPointer  string
}

type PlmnId struct {
	Mcc string
	Mnc string
}

type PlmnSupport struct {
	PlmnId     PlmnId
	SNssaiList []Snssai
}

type Snssai struct {
	Sst int32
	Sd  string
}

var amfContext *AMFContext
var once sync.Once

func GetAMFContext() *AMFContext {
	once.Do(func() {
		amfContext = &AMFContext{
			SupportedFeatures: make(map[string]bool),
		}
		logger.CtxLog.Info("AMF Context initialized")
	})
	return amfContext
}

func (c *AMFContext) NewUEContext(ranUeNgapId int64) *UEContext {
	ue := &UEContext{
		RanUeNgapId: ranUeNgapId,
		AmfUeNgapId: c.allocateAmfUeNgapId(),
	}
	c.UeContexts.Store(ue.AmfUeNgapId, ue)
	logger.CtxLog.Infof("New UE Context created, AMF UE NGAP ID: %d", ue.AmfUeNgapId)
	return ue
}

func (c *AMFContext) GetUEContextByAmfUeNgapId(id int64) (*UEContext, bool) {
	value, ok := c.UeContexts.Load(id)
	if !ok {
		return nil, false
	}
	return value.(*UEContext), true
}

func (c *AMFContext) DeleteUEContext(amfUeNgapId int64) {
	c.UeContexts.Delete(amfUeNgapId)
	logger.CtxLog.Infof("UE Context deleted, AMF UE NGAP ID: %d", amfUeNgapId)
}

func (c *AMFContext) allocateAmfUeNgapId() int64 {

	return 1
}
