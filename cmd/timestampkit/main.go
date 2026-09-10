package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/relentlessworks/timestampkit/internal/api"
	"github.com/relentlessworks/timestampkit/internal/auth"
	"github.com/relentlessworks/timestampkit/internal/config"
)

func main() {
	cfg := config.Load()

	authSvc := auth.New(cfg.Secret)
	server := api.NewServer(authSvc)

	mux := server.Routes()

	fmt.Fprintf(os.Stderr, "timestampkit starting on %s\n", cfg.String())
	log.Fatal(http.ListenAndServe(cfg.Addr, mux))
}
