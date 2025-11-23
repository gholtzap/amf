package factory

import (
	"encoding/json"
	"os"

	"github.com/gavin/amf/internal/logger"
)

type Config struct {
	Info          *Info          `json:"info"`
	Configuration *Configuration `json:"configuration"`
}

type Info struct {
	Version     string `json:"version"`
	Description string `json:"description"`
}

type Configuration struct {
	AmfName                         string        `json:"amfName"`
	NgapIpList                      []string      `json:"ngapIpList"`
	NgapPort                        int           `json:"ngapPort"`
	Sbi                             *Sbi          `json:"sbi"`
	ServiceNameList                 []string      `json:"serviceNameList"`
	ServedGuamiList                 []ServedGuami `json:"servedGuamiList"`
	SupportTaiList                  []SupportTai  `json:"supportTaiList"`
	PlmnSupportList                 []PlmnSupport `json:"plmnSupportList"`
	SupportDnnList                  []string      `json:"supportDnnList"`
	NfServices                      []NfService   `json:"nfServices"`
	Security                        *Security     `json:"security"`
	NetworkName                     *NetworkName  `json:"networkName"`
	TimeZone                        *TimeZone     `json:"timeZone"`
	T3502Value                      int           `json:"t3502Value"`
	T3512Value                      int           `json:"t3512Value"`
	Non3gppDeregistrationTimerValue int           `json:"non3gppDeregistrationTimerValue"`
	T3513                           *TimerValue   `json:"t3513"`
	T3522                           *TimerValue   `json:"t3522"`
	T3540                           *TimerValue   `json:"t3540"`
	T3550                           *TimerValue   `json:"t3550"`
	T3560                           *TimerValue   `json:"t3560"`
	T3565                           *TimerValue   `json:"t3565"`
	T3570                           *TimerValue   `json:"t3570"`
	NrfUri                          string        `json:"nrfUri"`
	UdmUri                          string        `json:"udmUri"`
	AusfUri                         string        `json:"ausfUri"`
	SmfUri                          string        `json:"smfUri"`
	DatabaseUri                     string        `json:"databaseUri"`
	DatabaseName                    string        `json:"databaseName"`
}

type Sbi struct {
	Scheme       string `json:"scheme"`
	RegisterIPv4 string `json:"registerIPv4"`
	BindingIPv4  string `json:"bindingIPv4"`
	Port         int    `json:"port"`
	Tls          *Tls   `json:"tls,omitempty"`
}

type Tls struct {
	Pem string `json:"pem"`
	Key string `json:"key"`
}

type ServedGuami struct {
	PlmnId      *PlmnId `json:"plmnId"`
	AmfId       string  `json:"amfId"`
	AmfRegionId string  `json:"amfRegionId,omitempty"`
	AmfSetId    string  `json:"amfSetId,omitempty"`
	AmfPointer  string  `json:"amfPointer,omitempty"`
}

type PlmnId struct {
	Mcc string `json:"mcc"`
	Mnc string `json:"mnc"`
}

type SupportTai struct {
	PlmnId *PlmnId `json:"plmnId"`
	Tac    string  `json:"tac"`
}

type PlmnSupport struct {
	PlmnId     *PlmnId  `json:"plmnId"`
	SNssaiList []Snssai `json:"sNssaiList"`
}

type Snssai struct {
	Sst int    `json:"sst"`
	Sd  string `json:"sd,omitempty"`
}

type NfService struct {
	ServiceName string   `json:"serviceName"`
	Version     []string `json:"version"`
}

type Security struct {
	IntegrityOrder []string `json:"integrityOrder"`
	CipheringOrder []string `json:"cipheringOrder"`
}

type NetworkName struct {
	Full  string `json:"full"`
	Short string `json:"short"`
}

type TimeZone struct {
	TimeZoneOffsetMinutes int  `json:"timeZoneOffsetMinutes"`
	DaylightSavingTime    int  `json:"daylightSavingTime"`
}

type TimerValue struct {
	Enable        bool `json:"enable"`
	ExpireTime    int  `json:"expireTime"`
	MaxRetryTimes int  `json:"maxRetryTimes"`
}

var amfConfig *Config

func InitConfigFactory(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return err
	}

	if config.Configuration.DatabaseUri == "" {
		if envUri := os.Getenv("MONGODB_URI"); envUri != "" {
			config.Configuration.DatabaseUri = envUri
			logger.CfgLog.Info("Using MongoDB URI from MONGODB_URI environment variable")
		}
	}

	if config.Configuration.DatabaseName == "" {
		if envDb := os.Getenv("MONGODB_DB_NAME"); envDb != "" {
			config.Configuration.DatabaseName = envDb
			logger.CfgLog.Info("Using MongoDB database name from MONGODB_DB_NAME environment variable")
		}
	}

	amfConfig = config
	logger.CfgLog.Infof("AMF Config loaded from %s", configPath)
	return nil
}

func GetConfig() *Config {
	return amfConfig
}
