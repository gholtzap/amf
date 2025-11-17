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
