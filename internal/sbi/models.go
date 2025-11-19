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
	Type      string `json:"type"`
	State     string `json:"state,omitempty"`
	TimeStamp string `json:"timeStamp,omitempty"`
	Supi      string `json:"supi,omitempty"`
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
