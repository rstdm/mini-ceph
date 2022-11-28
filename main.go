package main

import (
	"fmt"
	"github.com/rstdm/glados/internal/flags"
	"github.com/rstdm/glados/internal/logger"
	"github.com/rstdm/glados/internal/server"
	"os"
)

func main() {
	flagValues, err := flags.Parse()
	if err != nil {
		fmt.Printf("Failed to parse flags: %v\n", err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}

	log, loggerCleanup, err := logger.New(flagValues.UseProductionLogger)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}
	defer loggerCleanup()

	truncatedFlagValues := flagValues              // create a copy
	truncatedFlagValues.BearerToken = "<redacted>" // the token must not be logged
	log.Infow("Logging server configuration", "flagValues", truncatedFlagValues)

	serv, err := server.New(flagValues, log)
	if err != nil {
		log.Fatalw("Failed to start server", "err", err, "port", flagValues.Port)
		return
	}

	serv.Launch()
}
