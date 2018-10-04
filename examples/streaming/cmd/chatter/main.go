package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"

	chatter "goa.design/goa/examples/streaming"
	chattersvc "goa.design/goa/examples/streaming/gen/chatter"
)

// Server provides the means to start and stop a server.
type Server interface {
	// Start starts a server and sends any errors to the error channel.
	Start(errc chan error)
	// Stop stops a server.
	Stop() error
	// Addr returns the listen address.
	Addr() string
	// Type returns the server type (HTTP or gRPC)
	Type() string
}

func main() {
	// Define command line flags, add any other flag required to configure
	// the service.
	var (
		hostF     = flag.String("host", "localhost", "Server host (valid values: localhost)")
		domainF   = flag.String("domain", "", "Host domain name (overrides host domain and port specified in design)")
		httpPortF = flag.String("http-port", "", "HTTP port (used in conjunction with -- domain flag)")
		grpcPortF = flag.String("grpc-port", "", "gRPC port (used in conjunction with -- domain flag)")
		secureF   = flag.Bool("secure", false, "Use secure scheme (https or grpcs)")
		dbgF      = flag.Bool("debug", false, "Log request and response bodies")
	)
	flag.Parse()

	var (
		httpAddr string
		grpcAddr string
	)
	{
		if *domainF != "" {
			httpScheme := "http"
			if *secureF {
				httpScheme = "https"
			}
			httpPort := *httpPortF
			if httpPort == "" {
				httpPort = "80"
				if *secureF {
					httpPort = "443"
				}
			}
			httpAddr = httpScheme + "://" + *domainF + ":" + httpPort
			grpcScheme := "grpc"
			if *secureF {
				grpcScheme = "grpcs"
			}
			grpcPort := *grpcPortF
			if grpcPort == "" {
				grpcPort = "8080"
				if *secureF {
					grpcPort = "8443"
				}
			}
			grpcAddr = grpcScheme + "://" + *domainF + ":" + grpcPort
		}
	}

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

	var (
		addr string
		u    *url.URL
		svrs []Server
	)
	switch *hostF {
	case "localhost":
		if httpAddr != "" {
			addr = httpAddr
		} else {
			addr = "http://localhost:80"
		}
		u = parseAddr(addr)
		svrs = append(svrs, newHTTPServer(u.Scheme, u.Host, chatterEndpoints, logger, *dbgF))
		if grpcAddr != "" {
			addr = grpcAddr
		} else {
			addr = "grpc://localhost:8080"
		}
		u = parseAddr(addr)
		svrs = append(svrs, newGRPCServer(u.Scheme, u.Host, chatterEndpoints, logger, *dbgF))
	default:
		fmt.Fprintf(os.Stderr, "invalid host argument: %q (valid hosts: localhost", *hostF)
		os.Exit(1)
	}

	// Start the servers
	for _, s := range svrs {
		logger.Println("Starting " + s.Type() + " server at " + s.Addr())
		s.Start(errc)
	}

	// Wait for signal.
	logger.Printf("exiting (%v)", <-errc)
	for _, s := range svrs {
		logger.Println("Shutting down " + s.Type() + " server at " + s.Addr())
		s.Stop()
	}
	logger.Println("exited")
}

func parseAddr(addr string) *url.URL {
	u, err := url.Parse(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL %#v: %s", addr, err)
		os.Exit(1)
	}
	return u
}
