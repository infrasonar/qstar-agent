package main

import (
	"log"

	"github.com/infrasonar/go-libagent"
)

func main() {
	// Start collector
	log.Printf("Starting InfraSonar QStar Agent Collector v%s\n", version)

	// Initialize random
	libagent.RandInit()

	// Initialize Helper
	libagent.GetHelper()

	// Initialize the log helper
	InitLogHelper()

	// Set-up signal handler
	quit := make(chan bool)
	go libagent.SigHandler(quit)

	// Create Collector
	collector := libagent.NewCollector("qstar", version)

	// Create Asset
	asset := libagent.NewAsset(collector)
	// asset.Kind = "Linux"
	asset.Announce()

	// Create and plan checks
	checkQstar := libagent.Check{
		Key:             "qstar",
		Collector:       collector,
		Asset:           asset,
		IntervalEnv:     "CHECK_QSTAR_INTERVAL",
		DefaultInterval: 300,
		NoCount:         false,
		SetTimestamp:    false,
		Fn:              CheckQstar,
	}
	go checkQstar.Plan(quit)

	checkLog := libagent.Check{
		Key:             "log",
		Collector:       collector,
		Asset:           asset,
		IntervalEnv:     "CHECK_LOG_INTERVAL",
		DefaultInterval: 300,
		NoCount:         false,
		SetTimestamp:    false,
		Fn:              CheckLog,
	}
	go checkLog.Plan(quit)

	// Wait for quit
	<-quit
}
