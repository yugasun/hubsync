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
	Version = "dev"
)

func main() {
	// Initialize the logger (will be properly configured once config is loaded)
	utils.InitLogger("info", "")

	log.Info().
		Str("version", Version).
		Msg("Starting hubsync")

	// Run the application with improved dependency management
	if err := app.Run(Version); err != nil {
		log.Error().Err(err).Msg("Application failed")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("Application completed successfully")
}
