package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
)

func main() {
	cfg := common.LoadConfig()
	application, err := app.NewApplication(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()
	server := app.NewHTTPServer(application)

	address := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
	log.Printf("change-control-plane api listening on %s", address)
	if err := http.ListenAndServe(address, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
