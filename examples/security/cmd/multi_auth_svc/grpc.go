package main

import (
	"log"
	"net"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	secured_servicepb "goa.design/goa/examples/security/gen/grpc/secured_service"
	securedservicesvr "goa.design/goa/examples/security/gen/grpc/secured_service/server"
	securedservice "goa.design/goa/examples/security/gen/secured_service"
	"goa.design/goa/grpc/middleware"
	"google.golang.org/grpc"
)

func grpcServe(addr string, securedServiceEndpoints *securedservice.Endpoints, errc chan error, logger *log.Logger, debug bool) *grpc.Server {
	// Setup goa log adapter. Replace logger with your own using your
	// log package of choice. The goa.design/middleware/logging/...
	// packages define log adapters for common log packages.
	var (
		adapter middleware.Logger
	)
	{
		adapter = middleware.NewLogger(logger)
	}

	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to gRPC requests and
	// responses.
	var (
		securedServiceServer *securedservicesvr.Server
	)
	{
		securedServiceServer = securedservicesvr.New(securedServiceEndpoints)
	}

	// Initialize gRPC server with the middleware.
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			middleware.RequestID(),
			middleware.Log(adapter),
		)),
	)

	// Register the servers.
	secured_servicepb.RegisterSecuredServiceServer(srv, securedServiceServer)

	// Start gRPC server using default configuration, change the code to
	// configure the server as required by your service.
	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Fatalf("failed to listen: %v", err)
			errc <- err
		}
		logger.Printf("gRPC listening on %s", addr)
		errc <- srv.Serve(lis)
	}()

	return srv
}

func grpcStop(srv *grpc.Server) {
	srv.Stop()
}
