package context

import (
	"sync"
	"time"
)

// UEContext represents a UE context in AMF
type UEContext struct {
	mu sync.RWMutex

	// UE Identities
	Supi        string // Subscription Permanent Identifier (IMSI)
	Suci        string // Subscription Concealed Identifier
	Pei         string // Permanent Equipment Identifier (IMEI)
	Guti        *Guti  // 5G Globally Unique Temporary Identifier
	RequestedIdentityType uint8 // Identity type for T3570 retransmission

	// NGAP IDs
	RanUeNgapId int64  // RAN UE NGAP ID
	AmfUeNgapId int64  // AMF UE NGAP ID

	// Registration Information
	RegistrationType      uint8
	NgKsi                 int
	AuthenticationCtxId   string
	UeSecurityCapability  string

	// NAS Security Counters
	ULCount uint32
	DLCount uint32

	// Registration State
	RegistrationState RegistrationState

	// Location Info
	Tai             Tai    // Tracking Area Identity
	CellId          string

	// Security Context
	SecurityContext *SecurityContext

	// PDU Sessions
	PduSessions map[int32]*PduSessionContext // key: PDU Session ID

	// UE Capability
	UeCapability *UeCapability

	// Access and Mobility
	AccessType       AccessType
	CmState          CmState // Connection Management State
	RmState          RmState // Registration Management State

	// Timers
	T3502 *time.Timer // Deregistration attempt timer
	T3512 *time.Timer // Periodic registration update timer
	T3513 *time.Timer // Paging timer
	T3522 *time.Timer // Deregistration timer
	T3540 *time.Timer // Deregistration request timer
	T3550 *time.Timer // Registration Accept timer
	T3560 *time.Timer // Authentication Request timer
	T3565 *time.Timer // Security Mode Command timer
	T3570 *time.Timer // Identity Request timer

	// Timer retry counters
	T3550Counter int
	T3560Counter int
	T3565Counter int
	T3570Counter int
	T3540Counter int
	T3513Counter int
	T3522Counter int

	// Deregistration request state for T3540 retransmission
	DeregType              uint8
	DeregCause             uint8
	DeregReregRequired     bool

	// Re-authentication state
	IsReAuthenticating bool
	PreviousSecurityContext *SecurityContext

	// RAN Connection
	RanContext *RANContext

	// Subscription Data
	SubscriptionData *SubscriptionData
}

// RegistrationState represents UE registration state
type RegistrationState string

const (
	Deregistered       RegistrationState = "DEREGISTERED"
	Registered         RegistrationState = "REGISTERED"
	RegStateRegistering RegistrationState = "REGISTERING"
	RegStateRegistered  RegistrationState = "REGISTERED"
	RegStateDeregistered RegistrationState = "DEREGISTERED"
)

// AccessType represents access type (TS 23.501)
type AccessType string

const (
	AccessType3GPP    AccessType = "3GPP_ACCESS"
	AccessTypeNon3GPP AccessType = "NON_3GPP_ACCESS"
)

// CmState represents Connection Management State (TS 23.501 Section 5.3.2.3)
type CmState string

const (
	CmIdle      CmState = "CM_IDLE"
	CmConnected CmState = "CM_CONNECTED"
)

// RmState represents Registration Management State (TS 23.501 Section 5.3.2.2)
type RmState string

const (
	RmDeregistered RmState = "RM_DEREGISTERED"
	RmRegistered   RmState = "RM_REGISTERED"
)

// Tai represents Tracking Area Identity (TS 23.003)
type Tai struct {
	PlmnId PlmnId
	Tac    string // Tracking Area Code
}

// SecurityContext represents UE security context (TS 33.501)
type SecurityContext struct {
	Kseaf              []byte // Key from SEAF
	Kamf               []byte // AMF Key
	KnasInt            []byte // NAS Integrity Key
	KnasEnc            []byte // NAS Encryption Key
	NgKsi              int    // NAS Key Set Identifier
	IntegrityAlg       int    // NAS Integrity Algorithm
	CipheringAlg       int    // NAS Ciphering Algorithm
	IntegrityAlgorithm int    // Alias for IntegrityAlg
	CipheringAlgorithm int    // Alias for CipheringAlg
	Activated          bool   // Security context activation status
	SecurityCapability *UeSecurityCapability
}

// UeSecurityCapability represents UE security capabilities
type UeSecurityCapability struct {
	NrEncryptionAlgs []int // NR encryption algorithms
	NrIntegrityAlgs  []int // NR integrity algorithms
	EutraEncryptionAlgs []int // E-UTRA encryption algorithms
	EutraIntegrityAlgs  []int // E-UTRA integrity algorithms
}

// PduSessionContext represents PDU Session context
type PduSessionContext struct {
	PduSessionId   int32
	Dnn            string   // Data Network Name
	Snssai         Snssai   // S-NSSAI
	SessionAmbr    *Ambr    // Session AMBR
	QosFlows       map[int]*QosFlow
	State          PduSessionState
	AllocatedEbis  map[int32]int32 // EBI to ARP priority level mapping
	SmContextRef   string
	SmContextId    string
	AlwaysOn       bool
}

// PduSessionState represents PDU Session state
type PduSessionState string

const (
	PduSessionInactive PduSessionState = "INACTIVE"
	PduSessionActive   PduSessionState = "ACTIVE"
)

// Ambr represents Aggregate Maximum Bit Rate
type Ambr struct {
	Uplink   string // e.g., "100 Mbps"
	Downlink string // e.g., "200 Mbps"
}

// QosFlow represents QoS Flow
type QosFlow struct {
	QosFlowId     int
	FiveQi        int    // 5G QoS Identifier
	QosParameters *QosParameters
}

// QosParameters represents QoS parameters
type QosParameters struct {
	Priority       int
	PacketDelayBudget int
	PacketErrorRate   string
}

// UeCapability represents UE capabilities
type UeCapability struct {
	RatList                    []string
	UeRadioCapability          []byte
	UeRadioCapabilityForPaging []byte
}

// SubscriptionData represents UE subscription data
type SubscriptionData struct {
	Supi            string
	Nssai           []Snssai
	SubscribedUeAmbr *Ambr
	DnnConfigurations map[string]*DnnConfiguration
}

// DnnConfiguration represents DNN configuration
type DnnConfiguration struct {
	Dnn          string
	PduSessionTypes []string
	SscModes     []string
	IwkEpsInd    bool
	DefaultSnssai *Snssai
}

// NewPduSession creates a new PDU Session
func (ue *UEContext) NewPduSession(pduSessionId int32, dnn string, snssai Snssai) *PduSessionContext {
	ue.mu.Lock()
	defer ue.mu.Unlock()

	if ue.PduSessions == nil {
		ue.PduSessions = make(map[int32]*PduSessionContext)
	}

	session := &PduSessionContext{
		PduSessionId: pduSessionId,
		Dnn:          dnn,
		Snssai:       snssai,
		QosFlows:     make(map[int]*QosFlow),
		State:        PduSessionInactive,
	}

	ue.PduSessions[pduSessionId] = session
	return session
}

// GetPduSession retrieves PDU Session by ID
func (ue *UEContext) GetPduSession(pduSessionId int32) (*PduSessionContext, bool) {
	ue.mu.RLock()
	defer ue.mu.RUnlock()

	session, ok := ue.PduSessions[pduSessionId]
	return session, ok
}

func (ue *UEContext) DeletePduSession(pduSessionId int32) bool {
	ue.mu.Lock()
	defer ue.mu.Unlock()

	if _, ok := ue.PduSessions[pduSessionId]; !ok {
		return false
	}

	delete(ue.PduSessions, pduSessionId)
	return true
}

func (ue *UEContext) StopTimer(timer **time.Timer) {
	ue.mu.Lock()
	defer ue.mu.Unlock()

	if *timer != nil {
		(*timer).Stop()
		*timer = nil
	}
}

func (ue *UEContext) StopT3550() {
	ue.StopTimer(&ue.T3550)
	ue.T3550Counter = 0
}

func (ue *UEContext) StopT3560() {
	ue.StopTimer(&ue.T3560)
	ue.T3560Counter = 0
}

func (ue *UEContext) StopT3565() {
	ue.StopTimer(&ue.T3565)
	ue.T3565Counter = 0
}

func (ue *UEContext) StopT3512() {
	ue.StopTimer(&ue.T3512)
}

func (ue *UEContext) StopT3570() {
	ue.StopTimer(&ue.T3570)
	ue.T3570Counter = 0
}

func (ue *UEContext) StopT3540() {
	ue.StopTimer(&ue.T3540)
	ue.T3540Counter = 0
}

func (ue *UEContext) StopT3513() {
	ue.StopTimer(&ue.T3513)
	ue.T3513Counter = 0
}

func (ue *UEContext) StopT3522() {
	ue.StopTimer(&ue.T3522)
	ue.T3522Counter = 0
}

func (ue *UEContext) StopAllTimers() {
	ue.StopT3550()
	ue.StopT3560()
	ue.StopT3565()
	ue.StopT3570()
	ue.StopT3540()
	ue.StopT3512()
	ue.StopT3513()
	ue.StopT3522()
	ue.StopTimer(&ue.T3502)
}
