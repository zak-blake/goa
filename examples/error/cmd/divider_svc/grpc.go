package main

import (
	"log"
	"net"

	dividersvc "goa.design/goa/examples/error/gen/divider"
	dividerpb "goa.design/goa/examples/error/gen/grpc/divider"
	dividersvcsvr "goa.design/goa/examples/error/gen/grpc/divider/server"
	"google.golang.org/grpc"
)

func grpcServe(addr string, dividerEndpoints *dividersvc.Endpoints, errc chan error, logger *log.Logger, debug bool) *grpc.Server {
	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to gRPC requests and
	// responses.
	var (
		dividerServer *dividersvcsvr.Server
	)
	{
		dividerServer = dividersvcsvr.New(dividerEndpoints)
	}

	// Initialize gRPC server using default configuration.
	srv := grpc.NewServer()

	// Register the servers.
	dividerpb.RegisterDividerServer(srv, dividerServer)

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
