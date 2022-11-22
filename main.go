package main

import (
	"flag"
	"fmt"
	"github.com/rstdm/glados/internal/logger"
	"github.com/rstdm/glados/internal/server"
	"os"
)

func main() {
	var useProductionLogger bool
	var port int
	var bearerToken string

	flag.BoolVar(&useProductionLogger, "useProductionLogger", false, "Determines weather the logger "+
		"should produce json output or human readable output")
	flag.IntVar(&port, "port", 5000, "Port on which to serve http requests.")
	flag.StringVar(&bearerToken, "bearerToken", "", "BearerToken that is used to authorize all "+
		"requests. Every request will be accepted without authorization if the token is empty.")
	flag.Parse()

	log, loggerCleanup, err := logger.New(useProductionLogger)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		fmt.Println("Exiting.")
		os.Exit(1)
	}
	defer loggerCleanup()

	serv, err := server.New(port, bearerToken, log)
	if err != nil {
		log.Fatalw("Failed to start server", "err", err, "port", port)
		return
	}

	serv.Launch()
}
