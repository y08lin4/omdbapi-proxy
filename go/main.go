package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := LoadConfig()
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("start failed: %v", err)
	}

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("OMDb API manager listening on %s", cfg.ListenAddr)
	log.Printf("docs: http://localhost%s/docs", cfg.ListenAddr)
	log.Fatal(server.ListenAndServe())
}
