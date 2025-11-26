package sbi

import "github.com/gavin/amf/internal/context"

type UeContextCreateData struct {
	Supi string `json:"supi,omitempty"`
	Gpsi string `json:"gpsi,omitempty"`
	Pei  string `json:"pei,omitempty"`

	UdmGroupId string `json:"udmGroupId,omitempty"`

	AusfGroupId string `json:"ausfGroupId,omitempty"`

	Guami *Guami `json:"guami,omitempty"`

	PcfId            string   `json:"pcfId,omitempty"`
	PcfGroupId       string   `json:"pcfGroupId,omitempty"`
	PcfSetId         string   `json:"pcfSetId,omitempty"`
	RoutingIndicator string   `json:"routingIndicator,omitempty"`
	GroupList        []string `json:"groupList,omitempty"`

	IabOperationAllowed bool `json:"iabOperationAllowed,omitempty"`

	LocationInfo *UserLocation `json:"ueLocation,omitempty"`

	TimeZone string `json:"timeZone,omitempty"`

	RegRequest *RegistrationContextContainer `json:"regRequest,omitempty"`

	AccessType string `json:"accessType,omitempty"`

	RatType string `json:"ratType,omitempty"`

	InitialAmfName string `json:"initialAmfName,omitempty"`
}

type UeContextCreatedData struct {
	UeContext *UeContext `json:"ueContext,omitempty"`

	TargetId          *Guami              `json:"targetId,omitempty"`
	PduSessionList    []PduSessionContext `json:"pduSessionList,omitempty"`
	FailedSessionList []PduSessionContext `json:"failedSessionList,omitempty"`

	PcfReselectedInd bool `json:"pcfReselectedInd,omitempty"`

	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type UeContext struct {
	Supi                string    `json:"supi,omitempty"`
	SupiUnauthInd       bool      `json:"supiUnauthInd,omitempty"`
	GpsiList            []string  `json:"gpsiList,omitempty"`
	Pei                 string    `json:"pei,omitempty"`
	UdmGroupId          string    `json:"udmGroupId,omitempty"`
	AusfGroupId         string    `json:"ausfGroupId,omitempty"`
	RoutingIndicator    string    `json:"routingIndicator,omitempty"`
	GroupList           []string  `json:"groupList,omitempty"`
	IabOperationAllowed bool      `json:"iabOperationAllowed,omitempty"`
	SubRfsp             int32     `json:"subRfsp,omitempty"`
	SubUeAmbr           *Ambr     `json:"subUeAmbr,omitempty"`
	Smsf3GppAccessId    string    `json:"smsf3GppAccessId,omitempty"`
	SmsfNon3GppAccessId string    `json:"smsfNon3GppAccessId,omitempty"`
	SeafData            *SeafData `json:"seafData,omitempty"`
}

type PduSessionContext struct {
	PduSessionId     int32    `json:"pduSessionId"`
	SmContextRef     string   `json:"smContextRef,omitempty"`
	Snssai           *Snssai  `json:"sNssai,omitempty"`
	Dnn              string   `json:"dnn,omitempty"`
	AccessType       string   `json:"accessType,omitempty"`
	AllocatedEbiList []string `json:"allocatedEbiList,omitempty"`
	HsmfId           string   `json:"hsmfId,omitempty"`
	VsmfId           string   `json:"vsmfId,omitempty"`
	NsInstance       string   `json:"nsInstance,omitempty"`
}

type RegistrationContextContainer struct {
	UeContext        *UeContext       `json:"ueContext,omitempty"`
	LocalTimeZone    string           `json:"localTimeZone,omitempty"`
	AnType           string           `json:"anType,omitempty"`
	AnN2ApId         int64            `json:"anN2ApId,omitempty"`
	RanNodeId        *GlobalRanNodeId `json:"ranNodeId,omitempty"`
	InitialAmfName   string           `json:"initialAmfName,omitempty"`
	UserLocation     *UserLocation    `json:"userLocation,omitempty"`
	RrcEstCause      string           `json:"rrcEstCause,omitempty"`
	UeContextRequest string           `json:"ueContextRequest,omitempty"`
	InitialUeMessage []byte           `json:"initialUeMessage,omitempty"`
	AllowedNssai     *AllowedNssai    `json:"allowedNssai,omitempty"`
}

type UserLocation struct {
	NrLocation    *NrLocation    `json:"nrLocation,omitempty"`
	EutraLocation *EutraLocation `json:"eutraLocation,omitempty"`
	N3gaLocation  *N3gaLocation  `json:"n3gaLocation,omitempty"`
}

type NrLocation struct {
	Tai                     *Tai             `json:"tai"`
	Ncgi                    *Ncgi            `json:"ncgi"`
	IgnoreNcgi              bool             `json:"ignoreNcgi,omitempty"`
	AgeOfLocationInfo       int32            `json:"ageOfLocationInfo,omitempty"`
	UeLocationTimestamp     string           `json:"ueLocationTimestamp,omitempty"`
	GeographicalInformation string           `json:"geographicalInformation,omitempty"`
	GeodeticInformation     string           `json:"geodeticInformation,omitempty"`
	GlobalGnbId             *GlobalRanNodeId `json:"globalGnbId,omitempty"`
}

type EutraLocation struct {
	Tai                     *Tai             `json:"tai"`
	IgnoreTai               bool             `json:"ignoreTai,omitempty"`
	Ecgi                    *Ecgi            `json:"ecgi"`
	IgnoreEcgi              bool             `json:"ignoreEcgi,omitempty"`
	AgeOfLocationInfo       int32            `json:"ageOfLocationInfo,omitempty"`
	UeLocationTimestamp     string           `json:"ueLocationTimestamp,omitempty"`
	GeographicalInformation string           `json:"geographicalInformation,omitempty"`
	GeodeticInformation     string           `json:"geodeticInformation,omitempty"`
	GlobalNgenbId           *GlobalRanNodeId `json:"globalNgenbId,omitempty"`
}

type N3gaLocation struct {
	N3gppTai   *Tai   `json:"n3gppTai,omitempty"`
	N3IwfId    string `json:"n3IwfId,omitempty"`
	UeIpv4Addr string `json:"ueIpv4Addr,omitempty"`
	UeIpv6Addr string `json:"ueIpv6Addr,omitempty"`
	PortNumber int32  `json:"portNumber,omitempty"`
}

type Tai struct {
	PlmnId *PlmnId `json:"plmnId"`
	Tac    string  `json:"tac"`
}

type Ncgi struct {
	PlmnId   *PlmnId `json:"plmnId"`
	NrCellId string  `json:"nrCellId"`
}

type Ecgi struct {
	PlmnId      *PlmnId `json:"plmnId"`
	EutraCellId string  `json:"eutraCellId"`
}

type PlmnId struct {
	Mcc string `json:"mcc"`
	Mnc string `json:"mnc"`
}

type Snssai struct {
	Sst int32  `json:"sst"`
	Sd  string `json:"sd,omitempty"`
}

type Guami struct {
	PlmnId *PlmnId `json:"plmnId"`
	AmfId  string  `json:"amfId"`
}

type Ambr struct {
	Uplink   string `json:"uplink"`
	Downlink string `json:"downlink"`
}

type SeafData struct {
	NgKsi  *NgKsi `json:"ngKsi"`
	KeyAmf string `json:"keyAmf,omitempty"`
}

type NgKsi struct {
	Tsc      string `json:"tsc"`
	KsiValue int32  `json:"ksi"`
}

type GlobalRanNodeId struct {
	PlmnId  *PlmnId `json:"plmnId"`
	N3IwfId string  `json:"n3IwfId,omitempty"`
	GNbId   *GNbId  `json:"gNbId,omitempty"`
	NgeNbId string  `json:"ngeNbId,omitempty"`
}

type GNbId struct {
	BitLength int32  `json:"bitLength"`
	GNBValue  string `json:"gNBValue"`
}

type AllowedNssai struct {
	AllowedSnssaiList []Snssai `json:"allowedSnssaiList"`
	AccessType        string   `json:"accessType,omitempty"`
}

type UEContextRelease struct {
	Supi                string     `json:"supi,omitempty"`
	UnauthenticatedSupi bool       `json:"unauthenticatedSupi,omitempty"`
	NgapCause           *NgApCause `json:"ngapCause"`
}

type NgApCause struct {
	Group int32 `json:"group"`
	Value int32 `json:"value"`
}

type ProblemDetails struct {
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Cause    string `json:"cause,omitempty"`
}

type N1N2MessageTransferReqData struct {
	N1MessageContainer *N1MessageContainer `json:"n1MessageContainer,omitempty"`
	N2InfoContainer    *N2InfoContainer    `json:"n2InfoContainer,omitempty"`
	MtData             *RefToBinaryData    `json:"mtData,omitempty"`
	SkipInd            bool                `json:"skipInd,omitempty"`
	LastMsgIndication  bool                `json:"lastMsgIndication,omitempty"`
	PduSessionId       int32               `json:"pduSessionId,omitempty"`
	LcsCorrelationId   string              `json:"lcsCorrelationId,omitempty"`
	Ppi                int32               `json:"ppi,omitempty"`
	Arp5qi             int32               `json:"arp5qi,omitempty"`
}

type N1MessageContainer struct {
	N1MessageClass   string           `json:"n1MessageClass"`
	N1MessageContent *RefToBinaryData `json:"n1MessageContent"`
	NfId             string           `json:"nfId,omitempty"`
}

type N2InfoContainer struct {
	N2InformationClass string           `json:"n2InformationClass"`
	SmInfo             *N2SmInformation `json:"smInfo,omitempty"`
	RanInfo            *N2RanInformation `json:"ranInfo,omitempty"`
	NrppaInfo          *N2NrppaInformation `json:"nrppaInfo,omitempty"`
	PwsInfo            *N2PwsInformation `json:"pwsInfo,omitempty"`
	NfId               string           `json:"nfId,omitempty"`
}

type N2SmInformation struct {
	PduSessionId int32            `json:"pduSessionId"`
	N2InfoContent *RefToBinaryData `json:"n2InfoContent"`
	SNssai       *Snssai          `json:"sNssai,omitempty"`
}

type N2RanInformation struct {
	N2InfoContent *RefToBinaryData `json:"n2InfoContent"`
}

type N2NrppaInformation struct {
	NrppaPdu         *RefToBinaryData `json:"nrppaPdu"`
	NfId             string           `json:"nfId,omitempty"`
}

type N2PwsInformation struct {
	PwsContainer     *RefToBinaryData `json:"pwsContainer"`
	NgapMessageType  int32            `json:"ngapMessageType,omitempty"`
}

type RefToBinaryData struct {
	ContentId string `json:"contentId"`
}

type N1N2MessageTransferRspData struct {
	Cause             string `json:"cause"`
	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type N1N2MessageTransferError struct {
	Error             *ProblemDetails                 `json:"error"`
	PwsErrorInfo      *PwsErrorData                   `json:"pwsErrorInfo,omitempty"`
}

type PwsErrorData struct {
	NgapMessageType   int32    `json:"ngapMessageType,omitempty"`
	FailedNgapData    []byte   `json:"failedNgapData,omitempty"`
}

func ToInternalTai(tai *Tai) context.Tai {
	if tai == nil {
		return context.Tai{}
	}
	return context.Tai{
		PlmnId: context.PlmnId{
			Mcc: tai.PlmnId.Mcc,
			Mnc: tai.PlmnId.Mnc,
		},
		Tac: tai.Tac,
	}
}

func ToInternalSnssai(snssai *Snssai) context.Snssai {
	if snssai == nil {
		return context.Snssai{}
	}
	return context.Snssai{
		Sst: snssai.Sst,
		Sd:  snssai.Sd,
	}
}

func ToSbiTai(tai context.Tai) *Tai {
	return &Tai{
		PlmnId: &PlmnId{
			Mcc: tai.PlmnId.Mcc,
			Mnc: tai.PlmnId.Mnc,
		},
		Tac: tai.Tac,
	}
}

func ToSbiSnssai(snssai context.Snssai) *Snssai {
	return &Snssai{
		Sst: snssai.Sst,
		Sd:  snssai.Sd,
	}
}

type AmfEventSubscription struct {
	EventList                     []AmfEvent `json:"eventList"`
	EventNotifyUri                string     `json:"eventNotifyUri"`
	NotifyCorrelationId           string     `json:"notifyCorrelationId"`
	NfId                          string     `json:"nfId"`
	SubsChangeNotifyUri           string     `json:"subsChangeNotifyUri,omitempty"`
	SubsChangeNotifyCorrelationId string     `json:"subsChangeNotifyCorrelationId,omitempty"`
	Supi                          string     `json:"supi,omitempty"`
	GroupId                       string     `json:"groupId,omitempty"`
	Gpsi                          string     `json:"gpsi,omitempty"`
	Pei                           string     `json:"pei,omitempty"`
	AnyUE                         bool       `json:"anyUE,omitempty"`
	Options                       *AmfEventMode `json:"options,omitempty"`
}

type AmfEvent struct {
	Type          string   `json:"type"`
	ImmediateFlag bool     `json:"immediateFlag,omitempty"`
	AreaList      []AmfEventArea `json:"areaList,omitempty"`
}

type AmfEventArea struct {
	PresenceInfo *PresenceInfo `json:"presenceInfo,omitempty"`
	SNssai       *Snssai       `json:"sNssai,omitempty"`
}

type PresenceInfo struct {
	PraId           string `json:"praId,omitempty"`
	PresenceState   string `json:"presenceState,omitempty"`
	TrackingAreaList []Tai `json:"trackingAreaList,omitempty"`
}

type AmfEventMode struct {
	Trigger          string `json:"trigger,omitempty"`
	MaxReports       int32  `json:"maxReports,omitempty"`
	Expiry           string `json:"expiry,omitempty"`
}

type AmfCreateEventSubscription struct {
	Subscription      *AmfEventSubscription `json:"subscription"`
	SupportedFeatures string                `json:"supportedFeatures,omitempty"`
}

type AmfCreatedEventSubscription struct {
	Subscription      *AmfEventSubscription `json:"subscription"`
	SubscriptionId    string                `json:"subscriptionId"`
	ReportList        []AmfEventReport      `json:"reportList,omitempty"`
	SupportedFeatures string                `json:"supportedFeatures,omitempty"`
}

type AmfEventReport struct {
	Type                string                 `json:"type"`
	State               *AmfEventState         `json:"state,omitempty"`
	TimeStamp           string                 `json:"timeStamp,omitempty"`
	Supi                string                 `json:"supi,omitempty"`
	Gpsi                string                 `json:"gpsi,omitempty"`
	Pei                 string                 `json:"pei,omitempty"`
	Location            *UserLocation          `json:"location,omitempty"`
	Timezone            string                 `json:"timezone,omitempty"`
	AccessTypeList      []string               `json:"accessTypeList,omitempty"`
	RmInfoList          []RmInfo               `json:"rmInfoList,omitempty"`
	CmInfoList          []CmInfo               `json:"cmInfoList,omitempty"`
	Reachability        string                 `json:"reachability,omitempty"`
	CommFailure         *CommunicationFailure  `json:"commFailure,omitempty"`
	LossOfConnectType   string                 `json:"lossOfConnectType,omitempty"`
	NumberOfUes         int32                  `json:"numberOfUes,omitempty"`
	AreaList            []AmfEventArea         `json:"areaList,omitempty"`
}

type AmfEventState struct {
	Active         bool  `json:"active"`
	RemainReports  int32 `json:"remainReports,omitempty"`
	RemainDuration int32 `json:"remainDuration,omitempty"`
}

type RmInfo struct {
	RmState    string `json:"rmState"`
	AccessType string `json:"accessType"`
}

type CmInfo struct {
	CmState    string `json:"cmState"`
	AccessType string `json:"accessType"`
}

type CommunicationFailure struct {
	NasReleaseCode string `json:"nasReleaseCode,omitempty"`
	RanReleaseCode string `json:"ranReleaseCode,omitempty"`
}

type AmfUpdateEventSubscriptionItem struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type AmfUpdateEventOptionItem struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type AmfUpdatedEventSubscription struct {
	Subscription *AmfEventSubscription `json:"subscription"`
	ReportList   []AmfEventReport      `json:"reportList,omitempty"`
}

type RequestLocInfo struct {
	Req5gsLoc         bool   `json:"req5gsLoc,omitempty"`
	ReqCurrentLoc     bool   `json:"reqCurrentLoc,omitempty"`
	ReqRatType        bool   `json:"reqRatType,omitempty"`
	ReqTimeZone       bool   `json:"reqTimeZone,omitempty"`
	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type ProvideLocInfo struct {
	CurrentLoc         bool          `json:"currentLoc,omitempty"`
	Location           *UserLocation `json:"location,omitempty"`
	AdditionalLocation *UserLocation `json:"additionalLocation,omitempty"`
	GeoInfo            *GeographicArea `json:"geoInfo,omitempty"`
	LocationAge        int32         `json:"locationAge,omitempty"`
	RatType            string        `json:"ratType,omitempty"`
	Timezone           string        `json:"timezone,omitempty"`
	SupportedFeatures  string        `json:"supportedFeatures,omitempty"`
	OldGuami           *Guami        `json:"oldGuami,omitempty"`
}

type GeographicArea struct {
	Point            *Point            `json:"point,omitempty"`
	PointUncertainty *PointUncertainty `json:"pointUncertainty,omitempty"`
	Polygon          []Point           `json:"polygon,omitempty"`
}

type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type PointUncertainty struct {
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Uncertainty float64 `json:"uncertainty"`
}

type UeContextInfo struct {
	SupportVoPS         bool   `json:"supportVoPS,omitempty"`
	SupportVoPSn3gpp    bool   `json:"supportVoPSn3gpp,omitempty"`
	LastActTime         string `json:"lastActTime,omitempty"`
	AccessType          string `json:"accessType,omitempty"`
	RatType             string `json:"ratType,omitempty"`
	SupportedFeatures   string `json:"supportedFeatures,omitempty"`
}

type UeContextInfoClass string

const (
	UeContextInfoClassTADS UeContextInfoClass = "TADS"
)

type EnableUeReachabilityReqData struct {
	Reachability      string        `json:"reachability"`
	SupportedFeatures string        `json:"supportedFeatures,omitempty"`
	OldGuami          *Guami        `json:"oldGuami,omitempty"`
	ExtBufSupport     bool          `json:"extBufSupport,omitempty"`
	QosFlowInfoList   []QosFlowInfo `json:"qosFlowInfoList,omitempty"`
	PduSessionId      int32         `json:"pduSessionId,omitempty"`
}

type EnableUeReachabilityRspData struct {
	Reachability      string `json:"reachability"`
	SupportedFeatures string `json:"supportedFeatures,omitempty"`
}

type QosFlowInfo struct {
	Qfi        int32  `json:"qfi"`
	Ppi        int32  `json:"ppi,omitempty"`
	FiveQi     int32  `json:"5qi,omitempty"`
	DlDataSize int32  `json:"dlDataSize,omitempty"`
}

type UeReachability string

const (
	UeReachabilityUNREACHABLE UeReachability = "UNREACHABLE"
	UeReachabilityREACHABLE   UeReachability = "REACHABLE"
	UeReachabilityREG_UPDATE  UeReachability = "REGULATORY_ONLY"
)

type EnableGroupReachabilityReqData struct {
	UeInfoList               []UeInfo                `json:"ueInfoList"`
	Tmgi                     *Tmgi                   `json:"tmgi"`
	ReachabilityNotifyUri    string                  `json:"reachabilityNotifyUri,omitempty"`
	MbsServiceAreaInfoList   []MbsServiceAreaInfo    `json:"mbsServiceAreaInfoList,omitempty"`
	Arp                      *Arp                    `json:"arp,omitempty"`
	FiveQi                   int32                   `json:"5qi,omitempty"`
	SupportedFeatures        string                  `json:"supportedFeatures,omitempty"`
}

type EnableGroupReachabilityRspData struct {
	UeConnectedList   []string `json:"ueConnectedList,omitempty"`
	SupportedFeatures string   `json:"supportedFeatures,omitempty"`
}

type UeInfo struct {
	UeList       []string `json:"ueList"`
	PduSessionId int32    `json:"pduSessionId,omitempty"`
}

type Tmgi struct {
	MbsServiceId string  `json:"mbsServiceId"`
	PlmnId       *PlmnId `json:"plmnId"`
}

type MbsServiceAreaInfo struct {
	MbsServiceAreaId string `json:"mbsServiceAreaId"`
	TaiList          []Tai  `json:"taiList,omitempty"`
}

type Arp struct {
	PriorityLevel          int32  `json:"priorityLevel"`
	PreemptionCapability   string `json:"preemptCap,omitempty"`
	PreemptionVulnerability string `json:"preemptVuln,omitempty"`
}

type ReachabilityNotificationData struct {
	ReachableUeList   []ReachableUeInfo `json:"reachableUeList,omitempty"`
	UnreachableUeList []string          `json:"unreachableUeList,omitempty"`
}

type ReachableUeInfo struct {
	UeList       []string      `json:"ueList"`
	UserLocation *UserLocation `json:"userLocation,omitempty"`
}

type MbsN2MessageTransferReqData struct {
	MbsSessionId       *MbsSessionId        `json:"mbsSessionId"`
	AreaSessionId      *AreaSessionId       `json:"areaSessionId,omitempty"`
	N2MbsSmInfo        *N2MbsSmInfo         `json:"n2MbsSmInfo"`
	SupportedFeatures  string               `json:"supportedFeatures,omitempty"`
	RanNodeIdList      []GlobalRanNodeId    `json:"ranNodeIdList,omitempty"`
	NotifyUri          string               `json:"notifyUri,omitempty"`
	NotifyCorrelationId string              `json:"notifyCorrelationId,omitempty"`
}

type MbsN2MessageTransferRspData struct {
	Result             string        `json:"result"`
	SupportedFeatures  string        `json:"supportedFeatures,omitempty"`
	FailureList        []RanFailure  `json:"failureList,omitempty"`
}

type N2MbsSmInfo struct {
	NgapIeType string           `json:"ngapIeType"`
	NgapData   *RefToBinaryData `json:"ngapData"`
}

type RanFailure struct {
	RanId                *GlobalRanNodeId     `json:"ranId"`
	RanFailureCause      *NgApCause           `json:"ranFailureCause,omitempty"`
	RanFailureIndication string               `json:"ranFailureIndication,omitempty"`
}

type MbsSessionId struct {
	Tmgi        *Tmgi   `json:"tmgi,omitempty"`
	Ssm         *Ssm    `json:"ssm,omitempty"`
	Nid         string  `json:"nid,omitempty"`
}

type AreaSessionId int32

type Ssm struct {
	SourceIpAddr  string `json:"sourceIpAddr"`
	DestIpAddr    string `json:"destIpAddr"`
}

type MbsNotification struct {
	MbsSessionId        *MbsSessionId `json:"mbsSessionId"`
	AreaSessionId       *AreaSessionId `json:"areaSessionId,omitempty"`
	FailureList         []RanFailure  `json:"failureList"`
	NotifyCorrelationId string        `json:"notifyCorrelationId,omitempty"`
}

type MbsNgapIeType string

const (
	MbsNgapIeTypeMBS_SES_ACT_REQ    MbsNgapIeType = "MBS_SES_ACT_REQ"
	MbsNgapIeTypeMBS_SES_DEACT_REQ  MbsNgapIeType = "MBS_SES_DEACT_REQ"
	MbsNgapIeTypeMBS_SES_UPD_REQ    MbsNgapIeType = "MBS_SES_UPD_REQ"
)

type RanFailureIndication string

const (
	RanFailureIndicationNG_RAN_FAILURE_WITHOUT_RESTART RanFailureIndication = "NG_RAN_FAILURE_WITHOUT_RESTART"
	RanFailureIndicationNG_RAN_NOT_REACHABLE           RanFailureIndication = "NG_RAN_NOT_REACHABLE"
)

type N2InformationTransferReqData struct {
	TaiList                 []Tai              `json:"taiList,omitempty"`
	RatSelector             string             `json:"ratSelector,omitempty"`
	GlobalRanNodeList       []GlobalRanNodeId  `json:"globalRanNodeList,omitempty"`
	N2Information           *N2InfoContainer   `json:"n2Information,omitempty"`
	SupportedFeatures       string             `json:"supportedFeatures,omitempty"`
	N2NotifyCallbackUri     string             `json:"n2NotifyCallbackUri,omitempty"`
}

type N2InformationTransferRspData struct {
	Result            string              `json:"result"`
	PwsRspData        *PWSResponseData    `json:"pwsRspData,omitempty"`
	SupportedFeatures string              `json:"supportedFeatures,omitempty"`
	TssRspPerNgranList []TssRspPerNgran   `json:"tssRspPerNgranList,omitempty"`
}

type N2InformationTransferError struct {
	Error        *ProblemDetails `json:"error"`
	PwsErrorInfo *PWSErrorData   `json:"pwsErrorInfo,omitempty"`
}

type N2InfoNotification struct {
	Result            string              `json:"result"`
	PwsRspData        *PWSResponseData    `json:"pwsRspData,omitempty"`
	TssRspPerNgranList []TssRspPerNgran   `json:"tssRspPerNgranList,omitempty"`
	ErrorInfo         *ProblemDetails     `json:"errorInfo,omitempty"`
}

type PWSResponseData struct {
	NgapMessageType    int32  `json:"ngapMessageType,omitempty"`
	SerialNumber       int32  `json:"serialNumber,omitempty"`
	MessageIdentifier  int32  `json:"messageIdentifier,omitempty"`
	UnknownTaiList     []Tai  `json:"unknownTaiList,omitempty"`
	N2PwsSubMissInd    bool   `json:"n2PwsSubMissInd,omitempty"`
}

type PWSErrorData struct {
	NgapMessageType int32  `json:"ngapMessageType,omitempty"`
	FailedNgapData  []byte `json:"failedNgapData,omitempty"`
}

type TssRspPerNgran struct {
	NgranId          *GlobalRanNodeId  `json:"ngranId"`
	NgranFailureInfo *NgranFailureInfo `json:"ngranFailureInfo,omitempty"`
	TssContainer     *N2InfoContent    `json:"tssContainer,omitempty"`
}

type NgranFailureInfo struct {
	NgranCause *NgApCause `json:"ngranCause"`
}

type N2InfoContent struct {
	NgapData *RefToBinaryData `json:"ngapData"`
}

type N2InformationTransferResult string

const (
	N2InformationTransferResultINITIATED N2InformationTransferResult = "N2_INFO_TRANSFER_INITIATED"
)

type RatSelector string

const (
	RatSelectorEUTRA RatSelector = "E-UTRA"
	RatSelectorNR    RatSelector = "NR"
)

type AssignEbiData struct {
	PduSessionId    int32          `json:"pduSessionId"`
	ArpList         []Arp          `json:"arpList,omitempty"`
	ReleasedEbiList []int32        `json:"releasedEbiList,omitempty"`
	OldGuami        *Guami         `json:"oldGuami,omitempty"`
	ModifiedEbiList []EbiArpMapping `json:"modifiedEbiList,omitempty"`
}

type AssignedEbiData struct {
	PduSessionId    int32           `json:"pduSessionId"`
	AssignedEbiList []EbiArpMapping `json:"assignedEbiList,omitempty"`
	FailedArpList   []Arp           `json:"failedArpList,omitempty"`
	ReleasedEbiList []int32         `json:"releasedEbiList,omitempty"`
}

type AssignEbiError struct {
	Error           *ProblemDetails `json:"error"`
	FailedArpList   []Arp           `json:"failedArpList,omitempty"`
	ReleasedEbiList []int32         `json:"releasedEbiList,omitempty"`
}

type EbiArpMapping struct {
	EpsBearerId   int32 `json:"epsBearerId"`
	Arp           *Arp  `json:"arp"`
	RelSessionId  int32 `json:"relSessionId,omitempty"`
}

type EpsBearerId int32

type ContextCreateReqData struct {
	MbsSessionId             *MbsSessionId        `json:"mbsSessionId"`
	MbsServiceAreaInfoList   []MbsServiceAreaInfo `json:"mbsServiceAreaInfoList,omitempty"`
	MbsServiceArea           *MbsServiceArea      `json:"mbsServiceArea,omitempty"`
	N2MbsSmInfo              *N2MbsSmInfo         `json:"n2MbsSmInfo"`
	NotifyUri                string               `json:"notifyUri"`
	MaxResponseTime          int32                `json:"maxResponseTime,omitempty"`
	Snssai                   *Snssai              `json:"snssai"`
	MbsmfId                  string               `json:"mbsmfId,omitempty"`
	MbsmfServiceInstId       string               `json:"mbsmfServiceInstId,omitempty"`
	AssociatedSessionId      *AssociatedSessionId `json:"associatedSessionId,omitempty"`
}

type ContextCreateRspData struct {
	MbsSessionId     *MbsSessionId  `json:"mbsSessionId"`
	N2MbsSmInfoList  []N2MbsSmInfo  `json:"n2MbsSmInfoList,omitempty"`
	OperationStatus  string         `json:"operationStatus,omitempty"`
}

type ContextUpdateReqData struct {
	MbsServiceArea         *MbsServiceArea      `json:"mbsServiceArea,omitempty"`
	MbsServiceAreaInfoList []MbsServiceAreaInfo `json:"mbsServiceAreaInfoList,omitempty"`
	N2MbsSmInfo            *N2MbsSmInfo         `json:"n2MbsSmInfo,omitempty"`
	RanIdList              []GlobalRanNodeId    `json:"ranIdList,omitempty"`
	NoNgapSignallingInd    bool                 `json:"noNgapSignallingInd,omitempty"`
	NotifyUri              string               `json:"notifyUri,omitempty"`
	MaxResponseTime        int32                `json:"maxResponseTime,omitempty"`
	N2MbsInfoChangeInd     bool                 `json:"n2MbsInfoChangeInd,omitempty"`
}

type ContextUpdateRspData struct {
	N2MbsSmInfoList []N2MbsSmInfo `json:"n2MbsSmInfoList,omitempty"`
	OperationStatus string        `json:"operationStatus,omitempty"`
}

type ContextStatusNotification struct {
	MbsSessionId     *MbsSessionId    `json:"mbsSessionId"`
	AreaSessionId    *AreaSessionId   `json:"areaSessionId,omitempty"`
	N2MbsSmInfoList  []N2MbsSmInfo    `json:"n2MbsSmInfoList,omitempty"`
	OperationEvents  []OperationEvent `json:"operationEvents,omitempty"`
	OperationStatus  string           `json:"operationStatus,omitempty"`
	ReleasedInd      bool             `json:"releasedInd,omitempty"`
}

type ContextStatusNotificationResponse struct {
	MbsSessionId    *MbsSessionId  `json:"mbsSessionId"`
	AreaSessionId   *AreaSessionId `json:"areaSessionId,omitempty"`
	N2MbsSmInfoList []N2MbsSmInfo  `json:"n2MbsSmInfoList,omitempty"`
}

type OperationEvent struct {
	OpEventType          string              `json:"opEventType"`
	AmfId                string              `json:"amfId,omitempty"`
	NgranFailureEventList []NgranFailureEvent `json:"ngranFailureEventList,omitempty"`
}

type NgranFailureEvent struct {
	NgranId                *GlobalRanNodeId `json:"ngranId"`
	NgranFailureIndication string           `json:"ngranFailureIndication,omitempty"`
}

type MbsServiceArea struct {
	NcgiList []Ncgi `json:"ncgiList,omitempty"`
	TaiList  []Tai  `json:"taiList,omitempty"`
}

type AssociatedSessionId struct {
	PduSessionId int32  `json:"pduSessionId,omitempty"`
	NsiId        string `json:"nsiId,omitempty"`
}

type OperationStatus string

const (
	OperationStatusMbsSessionStartComplete     OperationStatus = "MBS_SESSION_START_COMPLETE"
	OperationStatusMbsSessionStartIncomplete   OperationStatus = "MBS_SESSION_START_INCOMPLETE"
	OperationStatusMbsSessionUpdateComplete    OperationStatus = "MBS_SESSION_UPDATE_COMPLETE"
	OperationStatusMbsSessionUpdateIncomplete  OperationStatus = "MBS_SESSION_UPDATE_INCOMPLETE"
)

type OpEventType string

const (
	OpEventTypeAmfChange     OpEventType = "AMF_CHANGE"
	OpEventTypeNgRanEvent    OpEventType = "NG_RAN_EVENT"
)

type NgranFailureIndication string

const (
	NgranFailureIndicationNgRanRestartOrStart         NgranFailureIndication = "NG_RAN_RESTART_OR_START"
	NgranFailureIndicationNgRanFailureWithoutRestart  NgranFailureIndication = "NG_RAN_FAILURE_WITHOUT_RESTART"
	NgranFailureIndicationNgRanNotReachable           NgranFailureIndication = "NG_RAN_NOT_REACHABLE"
	NgranFailureIndicationNgRanRequiredRelease        NgranFailureIndication = "NG_RAN_REQUIRED_RELEASE"
)

type SearchedUeContext struct {
	UeContextId      string  `json:"ueContextId,omitempty"`
	Supi             string  `json:"supi,omitempty"`
	AmfUeNgapId      int64   `json:"amfUeNgapId,omitempty"`
	Pei              string  `json:"pei,omitempty"`
	AccessType       string  `json:"accessType,omitempty"`
	CmState          string  `json:"cmState,omitempty"`
	RmState          string  `json:"rmState,omitempty"`
	Tai              *Tai    `json:"tai,omitempty"`
	PduSessionCount  int     `json:"pduSessionCount,omitempty"`
}

type UeContextSearchResult struct {
	UeContexts []SearchedUeContext `json:"ueContexts"`
	TotalCount int                 `json:"totalCount"`
}

type UeContextTransferReqData struct {
	Reason            string               `json:"reason"`
	AccessType        string               `json:"accessType"`
	PlmnId            *PlmnId              `json:"plmnId,omitempty"`
	RegRequest        *N1MessageContainer  `json:"regRequest,omitempty"`
	SupportedFeatures string               `json:"supportedFeatures,omitempty"`
}

type UeContextTransferRspData struct {
	UeContext                   *UeContext     `json:"ueContext"`
	UeRadioCapability           *N2InfoContent `json:"ueRadioCapability,omitempty"`
	UeRadioCapabilityForPaging  *N2InfoContent `json:"ueRadioCapabilityForPaging,omitempty"`
	UeNbiotRadioCapability      *N2InfoContent `json:"ueNbiotRadioCapability,omitempty"`
	SupportedFeatures           string         `json:"supportedFeatures,omitempty"`
}

type TransferReason string

const (
	TransferReasonInitReg            TransferReason = "INIT_REG"
	TransferReasonMobiReg            TransferReason = "MOBI_REG"
	TransferReasonMobiRegUeValidated TransferReason = "MOBI_REG_UE_VALIDATED"
)

type UeRegStatusUpdateReqData struct {
	TransferStatus       string          `json:"transferStatus"`
	ToReleaseSessionList []int32         `json:"toReleaseSessionList,omitempty"`
	PcfReselectedInd     bool            `json:"pcfReselectedInd,omitempty"`
	SmfChangeInfoList    []SmfChangeInfo `json:"smfChangeInfoList,omitempty"`
	AnalyticsNotUsedList []string        `json:"analyticsNotUsedList,omitempty"`
}

type UeRegStatusUpdateRspData struct {
	RegStatusTransferComplete bool `json:"regStatusTransferComplete"`
}

type SmfChangeInfo struct {
	PduSessionIdList []int32 `json:"pduSessionIdList"`
	SmfChangeInd     string  `json:"smfChangeInd"`
}

type UeContextTransferStatus string

const (
	UeContextTransferStatusTransferred    UeContextTransferStatus = "TRANSFERRED"
	UeContextTransferStatusNotTransferred UeContextTransferStatus = "NOT_TRANSFERRED"
)

type SmfChangeIndication string

const (
	SmfChangeIndicationChanged SmfChangeIndication = "CHANGED"
	SmfChangeIndicationRemoved SmfChangeIndication = "REMOVED"
)

type UeContextRelocateData struct {
	UeContext                *UeContext         `json:"ueContext"`
	TargetId                 *NgRanTargetId     `json:"targetId"`
	SourceToTargetData       *N2InfoContent     `json:"sourceToTargetData"`
	ForwardRelocationRequest *RefToBinaryData   `json:"forwardRelocationRequest"`
	PduSessionList           []N2SmInformation  `json:"pduSessionList,omitempty"`
	UeRadioCapability        *N2InfoContent     `json:"ueRadioCapability,omitempty"`
	NgapCause                *NgApCause         `json:"ngapCause,omitempty"`
	SupportedFeatures        string             `json:"supportedFeatures,omitempty"`
}

type UeContextRelocatedData struct {
	UeContext *UeContext `json:"ueContext"`
}

type NgRanTargetId struct {
	RanNodeId *GlobalRanNodeId `json:"ranNodeId"`
	Tai       *Tai             `json:"tai"`
}

type UeContextCancelRelocateData struct {
	Supi                     string           `json:"supi,omitempty"`
	RelocationCancelRequest  *RefToBinaryData `json:"relocationCancelRequest"`
}

type UeN1N2InfoSubscriptionCreateData struct {
	N2InformationClass  string  `json:"n2InformationClass,omitempty"`
	N2NotifyCallbackUri string  `json:"n2NotifyCallbackUri,omitempty"`
	N1MessageClass      string  `json:"n1MessageClass,omitempty"`
	N1NotifyCallbackUri string  `json:"n1NotifyCallbackUri,omitempty"`
	NfId                string  `json:"nfId,omitempty"`
	SupportedFeatures   string  `json:"supportedFeatures,omitempty"`
	OldGuami            *Guami  `json:"oldGuami,omitempty"`
}

type UeN1N2InfoSubscriptionCreatedData struct {
	N1n2NotifySubscriptionId string `json:"n1n2NotifySubscriptionId"`
	SupportedFeatures        string `json:"supportedFeatures,omitempty"`
}

type SubscriptionData struct {
	AmfStatusUri string   `json:"amfStatusUri"`
	GuamiList    []Guami  `json:"guamiList,omitempty"`
}

type AmfStatusChangeNotification struct {
	AmfStatusInfoList []AmfStatusInfo `json:"amfStatusInfoList"`
}

type AmfStatusInfo struct {
	GuamiList          []Guami `json:"guamiList"`
	StatusChange       string  `json:"statusChange,omitempty"`
	TargetAmfRemoval   string  `json:"targetAmfRemoval,omitempty"`
	TargetAmfFailure   string  `json:"targetAmfFailure,omitempty"`
}

type NonUeN2InfoSubscriptionCreateData struct {
	GlobalRanNodeList  []GlobalRanNodeId `json:"globalRanNodeList,omitempty"`
	AnTypeList         []string          `json:"anTypeList,omitempty"`
	N2InformationClass string            `json:"n2InformationClass"`
	N2NotifyCallbackUri string           `json:"n2NotifyCallbackUri"`
	NfId               string            `json:"nfId,omitempty"`
	SupportedFeatures  string            `json:"supportedFeatures,omitempty"`
	NotifCorrelationId string            `json:"notifCorrelationId,omitempty"`
}

type NonUeN2InfoSubscriptionCreatedData struct {
	N2NotifySubscriptionId string `json:"n2NotifySubscriptionId"`
	SupportedFeatures      string `json:"supportedFeatures,omitempty"`
	N2InformationClass     string `json:"n2InformationClass,omitempty"`
}

type N2InformationNotification struct {
	N2NotifySubscriptionId string           `json:"n2NotifySubscriptionId"`
	N2InfoContainer        *N2InfoContainer `json:"n2InfoContainer,omitempty"`
	ToReleaseSessionList   []int32          `json:"toReleaseSessionList,omitempty"`
	LcsCorrelationId       string           `json:"lcsCorrelationId,omitempty"`
	NotifyCorrelationId    string           `json:"notifyCorrelationId,omitempty"`
	GlobalRanNodeId        *GlobalRanNodeId `json:"globalRanNodeId,omitempty"`
	AnType                 string           `json:"anType,omitempty"`
}

type RequestPosInfo struct {
	LcsClientType           string   `json:"lcsClientType,omitempty"`
	LcsLocation             string   `json:"lcsLocation,omitempty"`
	Supi                    string   `json:"supi,omitempty"`
	Gpsi                    string   `json:"gpsi,omitempty"`
	Priority                string   `json:"priority,omitempty"`
	LcsServiceType          string   `json:"lcsServiceType,omitempty"`
	VelocityRequested       string   `json:"velocityRequested,omitempty"`
	LcsSupportedGADShapes   string   `json:"lcsSupportedGADShapes,omitempty"`
	LocationNotificationUri string   `json:"locationNotificationUri,omitempty"`
	SupportedFeatures       string   `json:"supportedFeatures,omitempty"`
	OldGuami                *Guami   `json:"oldGuami,omitempty"`
	Pei                     string   `json:"pei,omitempty"`
	HgmlcCallBackURI        string   `json:"hgmlcCallBackURI,omitempty"`
	LdrType                 string   `json:"ldrType,omitempty"`
	AfID                    string   `json:"afID,omitempty"`
	ScheduledLocTime        string   `json:"scheduledLocTime,omitempty"`
	ReliableLocReq          bool     `json:"reliableLocReq,omitempty"`
}

type CancelPosInfo struct {
	Supi                      string `json:"supi"`
	HgmlcCallBackURI          string `json:"hgmlcCallBackURI"`
	LdrReference              string `json:"ldrReference"`
	ServingLMFIdentification  string `json:"servingLMFIdentification,omitempty"`
	SupportedFeatures         string `json:"supportedFeatures,omitempty"`
}

type ProvidePosInfoExt struct {
	LocationEstimate       *GeographicArea `json:"locationEstimate,omitempty"`
	AccuracyFulfilmentInd  string          `json:"accuracyFulfilmentInd,omitempty"`
	AgeOfLocationEstimate  int32           `json:"ageOfLocationEstimate,omitempty"`
	VelocityEstimate       *VelocityEstimate `json:"velocityEstimate,omitempty"`
	CivicAddress           *CivicAddress   `json:"civicAddress,omitempty"`
	PositioningDataList    []PositioningMethodAndUsage `json:"positioningDataList,omitempty"`
	GnssPositioningDataList []GnssPositioningMethodAndUsage `json:"gnssPositioningDataList,omitempty"`
	Ecgi                   *Ecgi           `json:"ecgi,omitempty"`
	Ncgi                   *Ncgi           `json:"ncgi,omitempty"`
	TargetMmeName          string          `json:"targetMmeName,omitempty"`
	TargetMmeRealm         string          `json:"targetMmeRealm,omitempty"`
	UtranSrvccInd          bool            `json:"utranSrvccInd,omitempty"`
	UtranAccessInd         bool            `json:"utranAccessInd,omitempty"`
	AchievedQos            string          `json:"achievedQos,omitempty"`
	SupportedFeatures      string          `json:"supportedFeatures,omitempty"`
}

type VelocityEstimate struct {
	HSpeed         int32  `json:"hSpeed,omitempty"`
	Bearing        int32  `json:"bearing,omitempty"`
	VSpeed         int32  `json:"vSpeed,omitempty"`
	VDirection     string `json:"vDirection,omitempty"`
	HUncertainty   int32  `json:"hUncertainty,omitempty"`
	VUncertainty   int32  `json:"vUncertainty,omitempty"`
}

type CivicAddress struct {
	Country     string `json:"country,omitempty"`
	A1          string `json:"A1,omitempty"`
	A2          string `json:"A2,omitempty"`
	A3          string `json:"A3,omitempty"`
	A4          string `json:"A4,omitempty"`
	A5          string `json:"A5,omitempty"`
	A6          string `json:"A6,omitempty"`
	PRD         string `json:"PRD,omitempty"`
	POD         string `json:"POD,omitempty"`
	STS         string `json:"STS,omitempty"`
	HNO         string `json:"HNO,omitempty"`
	HNS         string `json:"HNS,omitempty"`
	LMK         string `json:"LMK,omitempty"`
	LOC         string `json:"LOC,omitempty"`
	NAM         string `json:"NAM,omitempty"`
	PC          string `json:"PC,omitempty"`
}

type PositioningMethodAndUsage struct {
	Method string `json:"method"`
	Mode   string `json:"mode,omitempty"`
	Usage  string `json:"usage,omitempty"`
}

type GnssPositioningMethodAndUsage struct {
	GnssId string `json:"gnssId"`
	Mode   string `json:"mode,omitempty"`
	Usage  string `json:"usage,omitempty"`
}
