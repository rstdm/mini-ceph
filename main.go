package main

import (
	"fmt"
	"github.com/rstdm/mini-ceph/internal/configuration"
	"github.com/rstdm/mini-ceph/internal/logger"
	"github.com/rstdm/mini-ceph/internal/server"
	"os"
)

func main() {
	flagValues, err := configuration.Parse()
	if err != nil {
		fmt.Printf("Failed to parse configuration: %v\n", err)
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

	truncatedFlagValues := flagValues                  // create a copy
	truncatedFlagValues.UserBearerToken = "<redacted>" // the token must not be logged
	truncatedFlagValues.ClusterBearerToken = "<redacted>"
	log.Infow("Logging server configuration", "flagValues", truncatedFlagValues)

	serv, err := server.New(flagValues, log)
	if err != nil {
		log.Fatalw("Failed to start server", "err", err, "port", flagValues.Port)
		return
	}

	serv.Launch()
}
