package main

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/yugasun/hubsync/internal/app"
	"github.com/yugasun/hubsync/internal/utils"
)

// Version is set during build via -ldflags
var Version = "0.1.0"

func main() {
	// Initialize the logger (will be properly configured once config is loaded)
	utils.InitLogger("info", "")

	// Run the application with improved dependency management
	if err := app.Run(Version); err != nil {
		log.Error().Err(err).Msg("Application failed")
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	log.Info().Msg("Application completed successfully")
}
