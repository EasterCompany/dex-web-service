package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/EasterCompany/dex-web-service/config"
	"github.com/EasterCompany/dex-web-service/endpoints"
	"github.com/EasterCompany/dex-web-service/utils"
)

const ServiceName = "dex-web-service"

var (
	version   = "0.0.0"
	branch    = "unknown"
	commit    = "unknown"
	buildDate = "unknown"
	buildYear = "unknown"
	buildHash = "unknown"
	arch      = "unknown"
)

func main() {
	utils.SetVersion(version, branch, commit, buildDate, buildYear, buildHash, arch)

	// Handle version/help commands first (before flag parsing)
	if len(os.Args) > 1 {
		arg := os.Args[1]
		switch arg {
		case "version", "--version", "-v":
			fmt.Println(utils.GetVersion().Str)
			os.Exit(0)
		case "help", "--help", "-h":
			fmt.Println("Dexter Web Service")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  dex-web-service              Start the web service")
			fmt.Println("  dex-web-service version      Display version information")
			os.Exit(0)
		}
	}

	// Define CLI flags
	flag.Parse()

	// Set the version for the service.
	utils.SetVersion(version, branch, commit, buildDate, buildYear, buildHash, arch)

	// Load the service map and find our own configuration.
	serviceMap, err := config.LoadServiceMap()
	if err != nil {
		log.Fatalf("FATAL: Could not load service-map.json: %v", err)
	}

	var selfConfig *config.ServiceEntry
	for _, service := range serviceMap.Services["be"] { // be for Backend Services
		if service.ID == ServiceName {
			selfConfig = &service
			break
		}
	}

	if selfConfig == nil {
		log.Fatalf("FATAL: Service '%s' not found in service-map.json under 'be' services. Shutting down.", ServiceName)
	}

	// Find local-cache-0 for caching
	var cacheConfig *config.ServiceEntry
	for _, service := range serviceMap.Services["os"] {
		if service.ID == "local-cache-0" {
			cacheConfig = &service
			break
		}
	}

	if cacheConfig != nil {
		pass := ""
		if cacheConfig.Credentials != nil {
			pass = cacheConfig.Credentials.Password
		}
		utils.InitRedis(fmt.Sprintf("%s:%s", cacheConfig.Domain, cacheConfig.Port), pass, 0)
	}

	// Get port from config, convert to integer.
	port, err := strconv.Atoi(selfConfig.Port)
	if err != nil {
		log.Fatalf("FATAL: Invalid port '%s' for service '%s' in service-map.json: %v", selfConfig.Port, ServiceName, err)
	}

	// Create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure HTTP server
	mux := http.NewServeMux()

	// Register handlers
	// /service endpoint is public (for health checks)
	mux.HandleFunc("/service", endpoints.ServiceHandler)
	// /metadata endpoint for link unfurling and content extraction
	mux.HandleFunc("/metadata", endpoints.MetadataHandler)
	// /webview endpoint for headless browser rendering
	mux.HandleFunc("/webview", endpoints.WebViewHandler)
	// /search endpoint for DuckDuckGo HTML search
	mux.HandleFunc("/search", endpoints.SearchHandler)
	// /scrape endpoint for full content extraction
	mux.HandleFunc("/scrape", endpoints.ScrapeHandler)
	// /open endpoint for protocol redirects (ssh, mosh, etc.)
	mux.HandleFunc("/open", endpoints.OpenHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux, // No CORS middleware needed initially, can add later
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server in a goroutine
	go func() {
		fmt.Printf("Starting %s on :%d\n", ServiceName, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server crashed: %v", err)
		}
	}()

	// Wait for shutdown signal (SIGTERM from systemd or SIGINT from Ctrl+C)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Mark service as ready
	utils.SetHealthStatus("OK", "Service is running normally")

	// Block here until signal received
	<-stop
	log.Println("Shutting down service...")

	// Graceful cleanup
	utils.SetHealthStatus("SHUTTING_DOWN", "Service is shutting down")
	cancel() // Signals any background goroutines to stop

	// Give the HTTP server 5 seconds to finish current requests
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 5*time.Second) // Use main ctx as parent
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	log.Println("Service exited cleanly")
}
