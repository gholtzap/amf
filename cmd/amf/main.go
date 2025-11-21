package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gavin/amf/internal/context"
	"github.com/gavin/amf/internal/database"
	"github.com/gavin/amf/internal/logger"
	"github.com/gavin/amf/internal/nas"
	"github.com/gavin/amf/internal/ngap"
	"github.com/gavin/amf/internal/sbi"
	"github.com/gavin/amf/pkg/factory"
)

var (
	configPath string
	version    = "1.0.0"
)

func init() {
	flag.StringVar(&configPath, "config", "config/amfcfg.json", "AMF configuration file path")
}

func main() {
	flag.Parse()

	logger.MainLog.Infof("AMF version: %s", version)
	logger.MainLog.Info("Starting 5G AMF (Access and Mobility Management Function)")

	if err := factory.InitConfigFactory(configPath); err != nil {
		logger.MainLog.Fatalf("Failed to load configuration: %v", err)
	}

	amfContext := context.GetAMFContext()
	config := factory.GetConfig()

	amfContext.Name = config.Configuration.AmfName
	logger.MainLog.Infof("AMF Name: %s", amfContext.Name)

	amfContext.InitializeNFClients(
		config.Configuration.NrfUri,
		config.Configuration.UdmUri,
		config.Configuration.AusfUri,
		config.Configuration.SmfUri,
	)

	if config.Configuration.DatabaseUri != "" && config.Configuration.DatabaseName != "" {
		dbClient, err := database.NewMongoDBClient(config.Configuration.DatabaseUri, config.Configuration.DatabaseName)
		if err != nil {
			logger.MainLog.Errorf("Failed to connect to MongoDB: %v", err)
			logger.MainLog.Warn("Continuing without database persistence")
		} else {
			ueRepo := database.NewUERepository(dbClient)
			subscriptionRepo := database.NewSubscriptionRepository(dbClient)

			amfContext.InitializeDatabase(dbClient, ueRepo, subscriptionRepo)
			logger.MainLog.Info("Database persistence enabled")

			if err := amfContext.RestoreFromDatabase(); err != nil {
				logger.MainLog.Errorf("Failed to restore data from database: %v", err)
			}
		}
	} else {
		logger.MainLog.Info("Database not configured, running in memory-only mode")
	}

	for _, servedGuami := range config.Configuration.ServedGuamiList {
		guami := context.Guami{
			PlmnId: context.PlmnId{
				Mcc: servedGuami.PlmnId.Mcc,
				Mnc: servedGuami.PlmnId.Mnc,
			},
			AmfId: servedGuami.AmfId,
		}
		amfContext.ServedGuami = append(amfContext.ServedGuami, guami)
	}

	for _, plmnSupport := range config.Configuration.PlmnSupportList {
		ps := context.PlmnSupport{
			PlmnId: context.PlmnId{
				Mcc: plmnSupport.PlmnId.Mcc,
				Mnc: plmnSupport.PlmnId.Mnc,
			},
		}
		for _, snssai := range plmnSupport.SNssaiList {
			ps.SNssaiList = append(ps.SNssaiList, context.Snssai{
				Sst: int32(snssai.Sst),
				Sd:  snssai.Sd,
			})
		}
		amfContext.PlmnSupportList = append(amfContext.PlmnSupportList, ps)
	}

	nasHandler := nas.NewHandler(amfContext)
	logger.MainLog.Info("NAS handler initialized")

	ngapHandler := ngap.NewHandler(amfContext)
	logger.MainLog.Info("NGAP handler initialized")

	ngapServer := ngap.NewServer(amfContext, ngapHandler)
	ngapHandler.SetServer(ngapServer)

	nasHandler.SetNGAPHandler(ngapHandler)
	ngapHandler.SetNASHandler(nasHandler)
	logger.MainLog.Info("NAS and NGAP handlers wired together")

	if len(config.Configuration.NgapIpList) > 0 {
		ngapAddr := config.Configuration.NgapIpList[0]
		ngapPort := config.Configuration.NgapPort
		if err := ngapServer.Listen(ngapAddr, ngapPort); err != nil {
			logger.MainLog.Fatalf("Failed to start NGAP server: %v", err)
		}

		go func() {
			if err := ngapServer.Serve(); err != nil {
				logger.MainLog.Fatalf("NGAP server failed: %v", err)
			}
		}()
		logger.MainLog.Infof("NGAP server started on %s:%d", ngapAddr, ngapPort)
	}

	sbiServer := sbi.NewServer(amfContext)

	go func() {
		if err := sbiServer.Run(); err != nil {
			logger.MainLog.Fatalf("SBI server failed: %v", err)
		}
	}()

	logger.MainLog.Info("AMF initialized successfully")

	if config.Configuration.NrfUri != "" {
		nfInstanceId := fmt.Sprintf("%s-%s", config.Configuration.AmfName, "instance-001")
		amfSetId := ""
		amfRegionId := ""

		if len(config.Configuration.ServedGuamiList) > 0 {
			if config.Configuration.ServedGuamiList[0].AmfSetId != "" {
				amfSetId = config.Configuration.ServedGuamiList[0].AmfSetId
			}
			if config.Configuration.ServedGuamiList[0].AmfRegionId != "" {
				amfRegionId = config.Configuration.ServedGuamiList[0].AmfRegionId
			}
		}

		if err := amfContext.RegisterWithNRF(
			nfInstanceId,
			config.Configuration.Sbi.Scheme,
			config.Configuration.Sbi.RegisterIPv4,
			amfSetId,
			amfRegionId,
			config.Configuration.Sbi.Port,
		); err != nil {
			logger.MainLog.Errorf("Failed to register with NRF: %v", err)
		} else {
			logger.MainLog.Infof("Successfully registered with NRF: %s", nfInstanceId)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.MainLog.Info("Shutting down AMF...")

	if err := amfContext.DeregisterFromNRF(); err != nil {
		logger.MainLog.Errorf("Failed to deregister from NRF: %v", err)
	} else {
		logger.MainLog.Info("Successfully deregistered from NRF")
	}

	if err := amfContext.Shutdown(); err != nil {
		logger.MainLog.Errorf("Failed to shutdown AMF context: %v", err)
	}

	fmt.Println("AMF stopped")
}
