package main

import (
	"fmt"
	"os"

	"github.com/yugasun/hubsync/internal/app"
	"github.com/yugasun/hubsync/internal/utils"
)

// Version is set during build via -ldflags
var Version = "0.1.0"

func main() {
	// Initialize the logger
	utils.InitLogger("info", "")

	// Run the application
	if err := app.Run(Version); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
