package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gobackend "github.com/zarz/spotiflac_android/go_backend"
	"github.com/zarz/spotiflac_android/go_backend/webgateway"
)

func main() {
	storageKey := strings.TrimSpace(os.Getenv("SPOTIFLAC_EXTENSION_STORAGE_KEY"))
	if storageKey == "" {
		log.Fatal("SPOTIFLAC_EXTENSION_STORAGE_KEY is required (base64-encoded 32-byte key)")
	}
	if err := gobackend.SetExtensionStorageMasterKey(storageKey); err != nil {
		log.Fatalf("invalid extension storage key: %v", err)
	}

	extensionsDir := envOrDefault("SPOTIFLAC_EXTENSIONS_DIR", "./extensions")
	dataDir := envOrDefault("SPOTIFLAC_DATA_DIR", "./data")
	if err := os.MkdirAll(extensionsDir, 0o700); err != nil {
		log.Fatalf("create extensions directory: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		log.Fatalf("create data directory: %v", err)
	}
	if err := gobackend.InitExtensionSystem(extensionsDir, dataDir); err != nil {
		log.Fatalf("initialize extension system: %v", err)
	}
	defer gobackend.CleanupExtensions()

	addr := envOrDefault("SPOTIFLAC_GATEWAY_ADDR", "127.0.0.1:8787")
	token := strings.TrimSpace(os.Getenv("SPOTIFLAC_GATEWAY_TOKEN"))
	server := &http.Server{
		Addr:              addr,
		Handler:           webgateway.NewHandler(webgateway.NewCoreBackend(), token),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       25 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownSignals
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("gateway shutdown: %v", err)
		}
	}()

	log.Printf("SpotiFLAC Web gateway listening on %s with %d metadata provider(s)", addr, gobackend.GetExtensionMetadataProviderCount())
	if token == "" {
		log.Printf("warning: SPOTIFLAC_GATEWAY_TOKEN is empty; gateway requests are not authenticated")
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("gateway server: %v", err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
