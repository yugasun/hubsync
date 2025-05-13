package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/internal/app"
	"github.com/yugasun/hubsync/internal/utils"
)

// Version information is set during build via -ldflags
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// Handle version flag
	for _, arg := range os.Args {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("hubsync version %s (built on %s)\n", version, buildDate)
			os.Exit(0)
		}
	}

	// Initialize the logger (will be properly configured once config is loaded)
	utils.InitLogger("info", "")

	log.Info().
		Str("version", version).
		Msg("Starting hubsync")

	// Run the application with improved dependency management
	if err := app.Run(version); err != nil {
		log.Error().Err(err).Msg("Application failed")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("Application completed successfully")
}
