package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	chatter "goa.design/goa/examples/streaming"
	chattersvc "goa.design/goa/examples/streaming/gen/chatter"
)

func main() {
	// Define command line flags, add any other flag required to configure
	// the service.
	var (
		httpAddrF = flag.String("http-listen", ":8080", "HTTP listen `address`")
		grpcAddrF = flag.String("grpc-listen", ":8081", "gRPC listen `address`")
		dbgF      = flag.Bool("debug", false, "Log request and response bodies")
	)
	flag.Parse()

	// Setup logger and goa log adapter. Replace logger with your own using
	// your log package of choice. The goa.design/middleware/logging/...
	// packages define log adapters for common log packages.
	var (
		logger *log.Logger
	)
	{
		logger = log.New(os.Stderr, "[chatter] ", log.Ltime)
	}

	// Create the structs that implement the services.
	var (
		chatterSvc chattersvc.Service
	)
	{
		chatterSvc = chatter.NewChatter(logger)
	}

	// Wrap the services in endpoints that can be invoked from other
	// services potentially running in different processes.
	var (
		chatterEndpoints *chattersvc.Endpoints
	)
	{
		chatterEndpoints = chattersvc.NewEndpoints(chatterSvc, chatter.ChatterBasicAuth, chatter.ChatterJWTAuth)
	}

	// Create channel used by both the signal handler and server goroutines
	// to notify the main goroutine when to stop the server.
	errc := make(chan error)

	// Setup interrupt handler. This optional step configures the process so
	// that SIGINT and SIGTERM signals cause the service to stop gracefully.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		errc <- fmt.Errorf("%s", <-c)
	}()
	httpSrvr := httpServe(*httpAddrF, chatterEndpoints, errc, logger, *dbgF)
	grpcSrvr := grpcServe(*grpcAddrF, chatterEndpoints, errc, logger, *dbgF)

	// Wait for signal.
	logger.Printf("exiting (%v)", <-errc)
	logger.Println("Shutting down HTTP server at " + *httpAddrF)
	httpStop(httpSrvr)
	logger.Println("Shutting down gRPC server at " + *grpcAddrF)
	grpcStop(grpcSrvr)
	logger.Println("exited")
}
