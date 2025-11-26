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
	IsEmergencyRegistration bool

	// NAS Security Counters
	ULCount uint32
	DLCount uint32

	// Registration State
	RegistrationState RegistrationState

	// Location Info
	Tai             Tai    // Tracking Area Identity
	LastTai         *Tai   // Previous Tracking Area Identity (for mobility detection)
	TaiList         []Tai  // List of allowed Tracking Area Identities
	ForbiddenTaiList []Tai // List of forbidden Tracking Area Identities
	ServiceAreaRestriction *ServiceAreaRestriction
	CellId          string
	CurrentRNA      uint16   // Current RAN Notification Area
	RNAList         []uint16 // List of configured RAN Notification Area IDs

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
	MicoMode         bool    // Mobile Initiated Connection Only mode

	// Timers
	T3502 *time.Timer // Deregistration attempt timer
	T3512 *time.Timer // Periodic registration update timer
	T3513 *time.Timer // Paging timer
	T3517 *time.Timer // Service Accept timer
	T3522 *time.Timer // Deregistration timer
	T3540 *time.Timer // Deregistration request timer
	T3550 *time.Timer // Registration Accept timer
	T3555 *time.Timer // Configuration Update Command timer
	T3560 *time.Timer // Authentication Request timer
	T3565 *time.Timer // Security Mode Command timer
	T3570 *time.Timer // Identity Request timer
	T3519 *time.Timer // Notification timer
	T3510 *time.Timer // Registration Request timer
	T3511 *time.Timer // Registration failure timer
	T3516 *time.Timer // 5GMM status timer
	T3520 *time.Timer // GUTI reallocation timer
	T3521 *time.Timer // Deregistration request timer (UE-initiated)
	T3525 *time.Timer // Identity response timer

	// Timer retry counters
	T3550Counter int
	T3555Counter int
	T3560Counter int
	T3565Counter int
	T3570Counter int
	T3540Counter int
	T3513Counter int
	T3517Counter int
	T3522Counter int
	T3519Counter int
	T3510Counter int
	T3511Counter int
	T3516Counter int
	T3520Counter int
	T3521Counter int
	T3525Counter int

	// Deregistration request state for T3540 retransmission
	DeregType              uint8
	DeregCause             uint8
	DeregReregRequired     bool

	// Service Accept state for T3517 retransmission
	ActivePduSessions      []uint8

	// Re-authentication state
	IsReAuthenticating bool
	PreviousSecurityContext *SecurityContext

	// RAN Connection
	RanContext *RANContext

	// Subscription Data
	SubscriptionData *SubscriptionData

	// eDRX (Extended Discontinuous Reception)
	EDrxParameters *EDrxParameters

	// PSM (Power Saving Mode)
	PSMParameters *PSMParameters

	PendingMessages []*PendingN1N2Message
}

type PendingN1N2Message struct {
	N1MessageContent []byte
	N2SmInfo         []byte
	PduSessionId     int32
	Timestamp        time.Time
}

type EDrxParameters struct {
	EDrxValue      uint8
	PagingTimeWindow uint8
	Enabled        bool
}

type PSMParameters struct {
	T3324Value uint8
	T3412ExtendedValue uint8
	Enabled bool
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

type ServiceAreaRestriction struct {
	RestrictionType string
	Areas           []AreaRestriction
	MaxNumOfTAs     int
}

type AreaRestriction struct {
	Tacs []string
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
	Dnn            string
	Snssai         Snssai
	SessionAmbr    *Ambr
	QosFlows       map[int]*QosFlow
	State          PduSessionState
	AllocatedEbis  map[int32]int32
	SmContextRef   string
	SmContextId    string
	AlwaysOn       bool
	SscMode        uint8
	PduSessionType uint8
	EapState       EapAuthState
	EapIdentifier  uint8
	EapRand        []byte
	EapAutn        []byte
	RequestMsg     []byte
	MptcpRequested bool
	MptcpIndication uint8
}

type PduSessionState string

const (
	PduSessionInactive PduSessionState = "INACTIVE"
	PduSessionActive   PduSessionState = "ACTIVE"
)

type EapAuthState string

const (
	EapStateNone       EapAuthState = "NONE"
	EapStateInProgress EapAuthState = "IN_PROGRESS"
	EapStateSuccess    EapAuthState = "SUCCESS"
	EapStateFailed     EapAuthState = "FAILED"
)

// Ambr represents Aggregate Maximum Bit Rate
type Ambr struct {
	Uplink   string // e.g., "100 Mbps"
	Downlink string // e.g., "200 Mbps"
}

// QosFlow represents QoS Flow
type QosFlow struct {
	QosFlowId     int
	FiveQi        int
	QosParameters *QosParameters
	ReflectiveQosIndicator bool
	ReflectiveQosTimer     uint16
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

func (session *PduSessionContext) AddQosFlow(qfi int, fiveQi int) *QosFlow {
	if session.QosFlows == nil {
		session.QosFlows = make(map[int]*QosFlow)
	}

	flow := &QosFlow{
		QosFlowId: qfi,
		FiveQi:    fiveQi,
	}

	session.QosFlows[qfi] = flow
	return flow
}

func (session *PduSessionContext) GetQosFlow(qfi int) (*QosFlow, bool) {
	flow, ok := session.QosFlows[qfi]
	return flow, ok
}

func (session *PduSessionContext) DeleteQosFlow(qfi int) bool {
	if _, ok := session.QosFlows[qfi]; !ok {
		return false
	}

	delete(session.QosFlows, qfi)
	return true
}

func (session *PduSessionContext) ModifyQosFlow(qfi int, fiveQi int, params *QosParameters) bool {
	flow, ok := session.QosFlows[qfi]
	if !ok {
		return false
	}

	if fiveQi > 0 {
		flow.FiveQi = fiveQi
	}

	if params != nil {
		flow.QosParameters = params
	}

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

func (ue *UEContext) StopT3555() {
	ue.StopTimer(&ue.T3555)
	ue.T3555Counter = 0
}

func (ue *UEContext) StopT3517() {
	ue.StopTimer(&ue.T3517)
	ue.T3517Counter = 0
}

func (ue *UEContext) StopT3519() {
	ue.StopTimer(&ue.T3519)
	ue.T3519Counter = 0
}

func (ue *UEContext) StopT3510() {
	ue.StopTimer(&ue.T3510)
	ue.T3510Counter = 0
}

func (ue *UEContext) StopT3511() {
	ue.StopTimer(&ue.T3511)
	ue.T3511Counter = 0
}

func (ue *UEContext) StopT3516() {
	ue.StopTimer(&ue.T3516)
	ue.T3516Counter = 0
}

func (ue *UEContext) StopT3520() {
	ue.StopTimer(&ue.T3520)
	ue.T3520Counter = 0
}

func (ue *UEContext) StopT3521() {
	ue.StopTimer(&ue.T3521)
	ue.T3521Counter = 0
}

func (ue *UEContext) StopT3525() {
	ue.StopTimer(&ue.T3525)
	ue.T3525Counter = 0
}

func (ue *UEContext) StopAllTimers() {
	ue.StopT3550()
	ue.StopT3555()
	ue.StopT3560()
	ue.StopT3565()
	ue.StopT3570()
	ue.StopT3540()
	ue.StopT3512()
	ue.StopT3513()
	ue.StopT3517()
	ue.StopT3522()
	ue.StopT3519()
	ue.StopT3510()
	ue.StopT3511()
	ue.StopT3516()
	ue.StopT3520()
	ue.StopT3521()
	ue.StopT3525()
	ue.StopTimer(&ue.T3502)
}

func (ue *UEContext) IsTaiInList(tai Tai) bool {
	ue.mu.RLock()
	defer ue.mu.RUnlock()

	for _, t := range ue.TaiList {
		if t.PlmnId.Mcc == tai.PlmnId.Mcc && t.PlmnId.Mnc == tai.PlmnId.Mnc && t.Tac == tai.Tac {
			return true
		}
	}
	return false
}

func (ue *UEContext) IsTaiForbidden(tai Tai) bool {
	ue.mu.RLock()
	defer ue.mu.RUnlock()

	for _, t := range ue.ForbiddenTaiList {
		if t.PlmnId.Mcc == tai.PlmnId.Mcc && t.PlmnId.Mnc == tai.PlmnId.Mnc && t.Tac == tai.Tac {
			return true
		}
	}
	return false
}

func (ue *UEContext) HasTaiChanged(newTai Tai) bool {
	ue.mu.RLock()
	defer ue.mu.RUnlock()

	return !(ue.Tai.PlmnId.Mcc == newTai.PlmnId.Mcc &&
		ue.Tai.PlmnId.Mnc == newTai.PlmnId.Mnc &&
		ue.Tai.Tac == newTai.Tac)
}

func (ue *UEContext) IsServiceAreaRestricted(tai Tai) bool {
	ue.mu.RLock()
	defer ue.mu.RUnlock()

	if ue.ServiceAreaRestriction == nil {
		return false
	}

	tacInRestrictedAreas := false
	for _, area := range ue.ServiceAreaRestriction.Areas {
		for _, tac := range area.Tacs {
			if tac == tai.Tac {
				tacInRestrictedAreas = true
				break
			}
		}
		if tacInRestrictedAreas {
			break
		}
	}

	if ue.ServiceAreaRestriction.RestrictionType == "ALLOWED_AREAS" {
		return !tacInRestrictedAreas
	} else if ue.ServiceAreaRestriction.RestrictionType == "NOT_ALLOWED_AREAS" {
		return tacInRestrictedAreas
	}

	return false
}
