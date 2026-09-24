package main

import (
	"easyllm/config"
	"easyllm/internal/server"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Tune Go runtime for minimal memory footprint on macOS:
	// Set target memory limit to 48MB and trigger GC at 50% heap growth.
	debug.SetMemoryLimit(48 * 1024 * 1024)
	debug.SetGCPercent(50)

	// Load .env file if present
	godotenv.Load()

	// Raise Go stdlib multipart limits so large batch imports (e.g. 3000 files)
	// don't fail with "multipart: message too large" due to default max parts.
	// See: GODEBUG=multipartmaxparts, multipartmaxheaders.
	gd := os.Getenv("GODEBUG")
	if !strings.Contains(gd, "multipartmaxparts=") {
		if gd != "" && !strings.HasSuffix(gd, ",") {
			gd += ","
		}
		gd += "multipartmaxparts=20000"
	}
	if !strings.Contains(gd, "multipartmaxheaders=") {
		if gd != "" && !strings.HasSuffix(gd, ",") {
			gd += ","
		}
		gd += "multipartmaxheaders=20000"
	}
	_ = os.Setenv("GODEBUG", gd)

	// Load configuration
	cfg := config.Load()

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.App.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize and run application
	app, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
