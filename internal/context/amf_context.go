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

	EventSubscriptions sync.Map

	N1N2Subscriptions sync.Map

	AMFStatusSubscriptions sync.Map

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

func (c *AMFContext) StoreEventSubscription(subscriptionId string, subscription interface{}) {
	c.EventSubscriptions.Store(subscriptionId, subscription)
	logger.CtxLog.Infof("Event subscription stored: %s", subscriptionId)
}

func (c *AMFContext) GetEventSubscription(subscriptionId string) (interface{}, bool) {
	return c.EventSubscriptions.Load(subscriptionId)
}

func (c *AMFContext) DeleteEventSubscription(subscriptionId string) {
	c.EventSubscriptions.Delete(subscriptionId)
	logger.CtxLog.Infof("Event subscription deleted: %s", subscriptionId)
}

func (c *AMFContext) AddN1N2Subscription(subscription *N1N2Subscription) {
	c.N1N2Subscriptions.Store(subscription.SubscriptionId, subscription)
	logger.CtxLog.Infof("N1N2 subscription stored: %s", subscription.SubscriptionId)
}

func (c *AMFContext) GetN1N2Subscription(subscriptionId string) (*N1N2Subscription, bool) {
	value, ok := c.N1N2Subscriptions.Load(subscriptionId)
	if !ok {
		return nil, false
	}
	return value.(*N1N2Subscription), true
}

func (c *AMFContext) DeleteN1N2Subscription(subscriptionId string) {
	c.N1N2Subscriptions.Delete(subscriptionId)
	logger.CtxLog.Infof("N1N2 subscription deleted: %s", subscriptionId)
}

func (c *AMFContext) StoreAMFStatusSubscription(subscriptionId string, subscription interface{}) {
	c.AMFStatusSubscriptions.Store(subscriptionId, subscription)
	logger.CtxLog.Infof("AMF status subscription stored: %s", subscriptionId)
}

func (c *AMFContext) GetAMFStatusSubscription(subscriptionId string) (interface{}, bool) {
	return c.AMFStatusSubscriptions.Load(subscriptionId)
}

func (c *AMFContext) DeleteAMFStatusSubscription(subscriptionId string) {
	c.AMFStatusSubscriptions.Delete(subscriptionId)
	logger.CtxLog.Infof("AMF status subscription deleted: %s", subscriptionId)
}

type N1N2Subscription struct {
	SubscriptionId      string
	UeContextId         string
	N1MessageClass      string
	N1NotifyCallbackUri string
	N2InformationClass  string
	N2NotifyCallbackUri string
	NfId                string
}
