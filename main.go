package main

import (
	"fmt"
	"github.com/rstdm/glados/internal/logger"
	"os"
)

func main() {
	log, loggerCleanup, err := logger.New(true)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}
	defer loggerCleanup()

	log.Info("Hello World!")
}
