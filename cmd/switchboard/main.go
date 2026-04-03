package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-switchboard/internal/server"
	"github.com/stockyard-dev/stockyard-switchboard/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9170"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./switchboard-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("switchboard: open database: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Switchboard — Self-hosted service registry\n")
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  ─────────────────────────────────\n\n")

	log.Printf("switchboard: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("switchboard: %v", err)
	}
}
