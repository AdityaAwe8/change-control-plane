package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
)

func main() {
	once := flag.Bool("once", false, "run a single worker heartbeat")
	interval := flag.Duration("interval", 15*time.Second, "worker heartbeat interval")
	flag.Parse()

	application, err := app.NewApplication(common.LoadConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	run := func() {
		identityCtx := context.Background()
		metrics, err := application.Metrics(identityCtx)
		if err != nil {
			log.Printf("worker heartbeat metrics unavailable: %v", err)
			return
		}
		log.Printf("worker heartbeat orgs=%d projects=%d services=%d integrations=%d", metrics.Organizations, metrics.Projects, metrics.Services, metrics.Integrations)
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
