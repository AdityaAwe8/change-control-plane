package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/workflows"
)

func main() {
	once := flag.Bool("once", false, "run a single worker heartbeat")
	interval := flag.Duration("interval", 15*time.Second, "worker heartbeat interval")
	flag.Parse()

	cfg := common.LoadConfig()
	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	controller := workflows.NewControlLoop(application, cfg.WorkerAutoAdvance)
	warnedNoToken := false
	run := func() {
		if cfg.WorkerToken == "" {
			if !warnedNoToken {
				log.Printf("worker control loop disabled: CCP_WORKER_TOKEN is not configured")
				warnedNoToken = true
			}
			return
		}
		identity, err := application.Auth.LoadIdentity(context.Background(), "Bearer "+cfg.WorkerToken, cfg.WorkerOrganizationID)
		if err != nil {
			log.Printf("worker control loop authentication failed: %v", err)
			return
		}
		summary, err := controller.RunOnce(auth.WithIdentity(context.Background(), identity))
		if err != nil {
			log.Printf("worker control loop failed: %v", err)
			return
		}
		log.Printf(
			"worker control loop actor=%s organization=%s scanned=%d claimed=%d started=%d completed=%d automated_decisions=%d integrations_scanned=%d syncs_claimed=%d syncs_completed=%d events_dispatched=%d failures=%d",
			identity.ActorLabel(),
			identity.ActiveOrganizationID,
			summary.Scanned,
			summary.Claimed,
			summary.Started,
			summary.Completed,
			summary.AutomatedDecisions,
			summary.IntegrationsScanned,
			summary.SyncsClaimed,
			summary.SyncsCompleted,
			summary.EventsDispatched,
			len(summary.Failures),
		)
		for _, failure := range summary.Failures {
			log.Printf("worker control loop item failure: %s", failure)
		}
	}

	run()
	if *once {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for range ticker.C {
		run()
	}
}
