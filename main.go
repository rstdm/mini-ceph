package main

import (
	"flag"
	"fmt"
	"github.com/rstdm/glados/internal/logger"
	"os"
)

func main() {
	useProductionLogger := true
	flag.BoolVar(&useProductionLogger, "useProductionLogger", false, "Determines weather the logger should produce json output or human readable output")
	flag.Parse()

	log, loggerCleanup, err := logger.New(useProductionLogger)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}
	defer loggerCleanup()

	log.Info("Hello World!")
}
