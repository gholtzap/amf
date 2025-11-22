package context

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gavin/amf/internal/consumer"
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

	NonUeN2InfoSubscriptions sync.Map

	NfId string

	NrfClient  *consumer.NRFClient
	UdmClient  *consumer.UDMClient
	AusfClient *consumer.AUSFClient
	SmfClient  *consumer.SMFClient

	NrfUri  string
	UdmUri  string
	AusfUri string
	SmfUri  string

	heartbeatCancel chan struct{}

	DbClient           DatabaseClient
	UeRepo             UERepository
	SubscriptionRepo   SubscriptionRepository
	persistenceEnabled bool

	amfUeNgapIdCounter int64
	tmsiCounter        uint32

	IsOverloaded   bool
	OverloadAction int

	mu sync.RWMutex
}

type DatabaseClient interface {
	Disconnect() error
}

type UERepository interface {
	Save(ue *UEContext) error
	FindByAmfUeNgapId(id int64) (*UEContext, error)
	FindAll() ([]*UEContext, error)
	Delete(amfUeNgapId int64) error
}

type SubscriptionRepository interface {
	SaveN1N2Subscription(subscriptionId, ueContextId, n1MessageClass, n1NotifyCallbackUri, n2InformationClass, n2NotifyCallbackUri, nfId string) error
	FindN1N2Subscription(subscriptionId string) (interface{}, error)
	FindAllN1N2Subscriptions() ([]interface{}, error)
	DeleteN1N2Subscription(subscriptionId string) error
	SaveEventSubscription(subscriptionId string, data map[string]interface{}) error
	FindAllEventSubscriptions() ([]interface{}, error)
	DeleteEventSubscription(subscriptionId string) error
	SaveAMFStatusSubscription(subscriptionId string, data map[string]interface{}) error
	FindAllAMFStatusSubscriptions() ([]interface{}, error)
	DeleteAMFStatusSubscription(subscriptionId string) error
}

type Guami struct {
	PlmnId      PlmnId
	AmfId       string
	AmfRegionId string
	AmfSetId    string
	AmfPointer  string
}

type Guti struct {
	PlmnId      PlmnId
	AmfRegionId string
	AmfSetId    string
	AmfPointer  string
	Tmsi        uint32
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
		RanUeNgapId:      ranUeNgapId,
		AmfUeNgapId:      c.allocateAmfUeNgapId(),
		SecurityContext:  &SecurityContext{},
		PduSessions:      make(map[int32]*PduSessionContext),
	}
	c.UeContexts.Store(ue.AmfUeNgapId, ue)
	logger.CtxLog.Infof("New UE Context created, AMF UE NGAP ID: %d", ue.AmfUeNgapId)

	if c.persistenceEnabled && c.UeRepo != nil {
		if err := c.UeRepo.Save(ue); err != nil {
			logger.CtxLog.Errorf("Failed to persist UE context: %v", err)
		}
	}

	return ue
}

func (c *AMFContext) GetUEContextByAmfUeNgapId(id int64) (*UEContext, bool) {
	value, ok := c.UeContexts.Load(id)
	if !ok {
		return nil, false
	}
	return value.(*UEContext), true
}

func (c *AMFContext) GetUEContextByGuti(guti *Guti) (*UEContext, bool) {
	var foundUe *UEContext
	c.UeContexts.Range(func(key, value interface{}) bool {
		ue := value.(*UEContext)
		if ue.Guti != nil &&
		   ue.Guti.PlmnId.Mcc == guti.PlmnId.Mcc &&
		   ue.Guti.PlmnId.Mnc == guti.PlmnId.Mnc &&
		   ue.Guti.AmfRegionId == guti.AmfRegionId &&
		   ue.Guti.AmfSetId == guti.AmfSetId &&
		   ue.Guti.AmfPointer == guti.AmfPointer &&
		   ue.Guti.Tmsi == guti.Tmsi {
			foundUe = ue
			return false
		}
		return true
	})
	if foundUe != nil {
		return foundUe, true
	}
	return nil, false
}

func (c *AMFContext) DeleteUEContext(amfUeNgapId int64) {
	c.UeContexts.Delete(amfUeNgapId)
	logger.CtxLog.Infof("UE Context deleted, AMF UE NGAP ID: %d", amfUeNgapId)

	if c.persistenceEnabled && c.UeRepo != nil {
		if err := c.UeRepo.Delete(amfUeNgapId); err != nil {
			logger.CtxLog.Errorf("Failed to delete UE context from database: %v", err)
		}
	}
}

func (c *AMFContext) allocateAmfUeNgapId() int64 {
	return atomic.AddInt64(&c.amfUeNgapIdCounter, 1)
}

func (c *AMFContext) allocateTmsi() uint32 {
	return atomic.AddUint32(&c.tmsiCounter, 1)
}

func (c *AMFContext) AllocateGuti() *Guti {
	if len(c.ServedGuami) == 0 {
		logger.CtxLog.Warn("No GUAMI configured, cannot allocate GUTI")
		return nil
	}

	guami := c.ServedGuami[0]
	tmsi := c.allocateTmsi()

	guti := &Guti{
		PlmnId: guami.PlmnId,
		AmfRegionId: guami.AmfRegionId,
		AmfSetId: guami.AmfSetId,
		AmfPointer: guami.AmfPointer,
		Tmsi: tmsi,
	}

	logger.CtxLog.Infof("Allocated GUTI: %+v", guti)
	return guti
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

	if c.persistenceEnabled && c.SubscriptionRepo != nil {
		if err := c.SubscriptionRepo.SaveN1N2Subscription(
			subscription.SubscriptionId,
			subscription.UeContextId,
			subscription.N1MessageClass,
			subscription.N1NotifyCallbackUri,
			subscription.N2InformationClass,
			subscription.N2NotifyCallbackUri,
			subscription.NfId,
		); err != nil {
			logger.CtxLog.Errorf("Failed to persist N1N2 subscription: %v", err)
		}
	}
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

	if c.persistenceEnabled && c.SubscriptionRepo != nil {
		if err := c.SubscriptionRepo.DeleteN1N2Subscription(subscriptionId); err != nil {
			logger.CtxLog.Errorf("Failed to delete N1N2 subscription from database: %v", err)
		}
	}
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

func (c *AMFContext) InitializeNFClients(nrfUri, udmUri, ausfUri, smfUri string) {
	c.NrfUri = nrfUri
	c.UdmUri = udmUri
	c.AusfUri = ausfUri
	c.SmfUri = smfUri

	if nrfUri != "" {
		c.NrfClient = consumer.NewNRFClient(nrfUri)
		logger.CtxLog.Infof("NRF client initialized with URI: %s", nrfUri)
	}

	if udmUri != "" {
		c.UdmClient = consumer.NewUDMClient(udmUri)
		logger.CtxLog.Infof("UDM client initialized with URI: %s", udmUri)
	}

	if ausfUri != "" {
		c.AusfClient = consumer.NewAUSFClient(ausfUri)
		logger.CtxLog.Infof("AUSF client initialized with URI: %s", ausfUri)
	}

	if smfUri != "" {
		c.SmfClient = consumer.NewSMFClient(smfUri)
		logger.CtxLog.Infof("SMF client initialized with URI: %s", smfUri)
	}
}

func (c *AMFContext) RegisterWithNRF(nfInstanceId, scheme, ipv4, amfSetId, amfRegionId string, port int) error {
	if c.NrfClient == nil {
		logger.CtxLog.Warn("NRF client not initialized, skipping registration")
		return nil
	}

	c.NfId = nfInstanceId

	profile := &consumer.NFProfile{
		NfInstanceId:  nfInstanceId,
		NfType:        "AMF",
		NfStatus:      "REGISTERED",
		HeartBeatTimer: 30,
		Ipv4Addresses: []string{ipv4},
		Capacity:      100,
		Priority:      1,
		AmfInfo: &consumer.AmfInfo{
			AmfSetId:    amfSetId,
			AmfRegionId: amfRegionId,
			GuamiList:   make([]consumer.GuamiInfo, 0),
			TaiList:     make([]consumer.Tai, 0),
		},
		NfServices: []consumer.NFService{
			{
				ServiceInstanceId: "namf-comm",
				ServiceName:       "namf-comm",
				Versions: []consumer.NFServiceVersion{
					{
						ApiVersionInUri: "v1",
						ApiFullVersion:  "1.0.0",
					},
				},
				Scheme:          scheme,
				NfServiceStatus: "REGISTERED",
				ApiPrefix:       fmt.Sprintf("%s://%s:%d", scheme, ipv4, port),
				Ipv4Addresses:   []string{ipv4},
			},
		},
	}

	for _, plmn := range c.PlmnSupportList {
		profile.PlmnList = append(profile.PlmnList, consumer.PlmnId{
			Mcc: plmn.PlmnId.Mcc,
			Mnc: plmn.PlmnId.Mnc,
		})

		for _, snssai := range plmn.SNssaiList {
			profile.SNssais = append(profile.SNssais, consumer.SNssai{
				Sst: int(snssai.Sst),
				Sd:  snssai.Sd,
			})
		}
	}

	for _, guami := range c.ServedGuami {
		profile.AmfInfo.GuamiList = append(profile.AmfInfo.GuamiList, consumer.GuamiInfo{
			PlmnId: consumer.PlmnId{
				Mcc: guami.PlmnId.Mcc,
				Mnc: guami.PlmnId.Mnc,
			},
			AmfId: guami.AmfId,
		})
	}

	_, err := c.NrfClient.RegisterNF(profile)
	if err != nil {
		return fmt.Errorf("failed to register with NRF: %w", err)
	}

	c.StartHeartbeat(nfInstanceId, 30*time.Second)

	return nil
}

func (c *AMFContext) StartHeartbeat(nfInstanceId string, interval time.Duration) {
	if c.NrfClient == nil {
		return
	}

	c.heartbeatCancel = make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.NrfClient.SendHeartbeat(nfInstanceId); err != nil {
					logger.CtxLog.Errorf("Failed to send heartbeat: %v", err)
				}
			case <-c.heartbeatCancel:
				logger.CtxLog.Info("Heartbeat stopped")
				return
			}
		}
	}()

	logger.CtxLog.Infof("Heartbeat started for NF: %s (interval: %v)", nfInstanceId, interval)
}

func (c *AMFContext) StopHeartbeat() {
	if c.heartbeatCancel != nil {
		close(c.heartbeatCancel)
	}
}

func (c *AMFContext) DeregisterFromNRF() error {
	if c.NrfClient == nil || c.NfId == "" {
		return nil
	}

	c.StopHeartbeat()

	if err := c.NrfClient.DeregisterNF(c.NfId); err != nil {
		return fmt.Errorf("failed to deregister from NRF: %w", err)
	}

	return nil
}

func (c *AMFContext) InitializeDatabase(dbClient DatabaseClient, ueRepo UERepository, subscriptionRepo SubscriptionRepository) {
	c.DbClient = dbClient
	c.UeRepo = ueRepo
	c.SubscriptionRepo = subscriptionRepo
	c.persistenceEnabled = true
	logger.CtxLog.Info("Database persistence enabled")
}

func (c *AMFContext) RestoreFromDatabase() error {
	if !c.persistenceEnabled || c.UeRepo == nil {
		logger.CtxLog.Info("Database persistence not enabled, skipping restoration")
		return nil
	}

	ueContexts, err := c.UeRepo.FindAll()
	if err != nil {
		return fmt.Errorf("failed to restore UE contexts: %w", err)
	}

	for _, ue := range ueContexts {
		c.UeContexts.Store(ue.AmfUeNgapId, ue)
	}

	logger.CtxLog.Infof("Restored %d UE contexts from database", len(ueContexts))
	return nil
}

func (c *AMFContext) PersistUEContext(ue *UEContext) error {
	if !c.persistenceEnabled || c.UeRepo == nil {
		return nil
	}

	if err := c.UeRepo.Save(ue); err != nil {
		return fmt.Errorf("failed to persist UE context: %w", err)
	}

	return nil
}

func (c *AMFContext) GetRANContextsByTAI(tai Tai) []*RANContext {
	ranList := make([]*RANContext, 0)

	c.RanContexts.Range(func(key, value interface{}) bool {
		ran := value.(*RANContext)
		for _, supportedTAI := range ran.SupportedTAList {
			if supportedTAI.Tai.PlmnId.Mcc == tai.PlmnId.Mcc &&
			   supportedTAI.Tai.PlmnId.Mnc == tai.PlmnId.Mnc &&
			   supportedTAI.Tai.Tac == tai.Tac {
				ranList = append(ranList, ran)
				break
			}
		}
		return true
	})

	return ranList
}

func (c *AMFContext) Shutdown() error {
	c.StopHeartbeat()

	if c.DbClient != nil {
		if err := c.DbClient.Disconnect(); err != nil {
			return fmt.Errorf("failed to disconnect from database: %w", err)
		}
	}

	logger.CtxLog.Info("AMF Context shutdown complete")
	return nil
}
