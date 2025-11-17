package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gavin/amf/internal/context"
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
	_ = nasHandler

	ngapHandler := ngap.NewHandler(amfContext)
	logger.MainLog.Info("NGAP handler initialized")
	_ = ngapHandler

	sbiServer := sbi.NewServer(amfContext)

	go func() {
		if err := sbiServer.Run(); err != nil {
			logger.MainLog.Fatalf("SBI server failed: %v", err)
		}
	}()

	logger.MainLog.Info("AMF initialized successfully")

	logger.MainLog.Info("NGAP server start pending implementation")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	logger.MainLog.Info("Shutting down AMF...")

	fmt.Println("AMF stopped")
}
