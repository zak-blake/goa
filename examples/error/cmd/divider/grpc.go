package main

import (
	"log"
	"net"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	dividersvc "goa.design/goa/examples/error/gen/divider"
	"goa.design/goa/examples/error/gen/grpc/divider/pb"
	dividersvcsvr "goa.design/goa/examples/error/gen/grpc/divider/server"
	"goa.design/goa/grpc/middleware"
	"google.golang.org/grpc"
)

// grpcsvr implements Server interface.
type grpcsvr struct {
	svr  *grpc.Server
	addr string
}

func newGRPCServer(scheme, host string, dividerEndpoints *dividersvc.Endpoints, logger *log.Logger, debug bool) Server {
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
		dividerServer *dividersvcsvr.Server
	)
	{
		dividerServer = dividersvcsvr.New(dividerEndpoints, nil)
	}

	// Initialize gRPC server with the middleware.
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			middleware.RequestID(),
			middleware.Log(adapter),
		)),
	)

	// Register the servers.
	pb.RegisterDividerServer(srv, dividerServer)

	return &grpcsvr{svr: srv, addr: host}
}

func (g *grpcsvr) Start(errc chan error) {
	go func() {
		lis, err := net.Listen("tcp", g.addr)
		if err != nil {
			errc <- err
		}
		errc <- g.svr.Serve(lis)
	}()
}

func (g *grpcsvr) Stop() error {
	g.svr.Stop()
	return nil
}

func (g *grpcsvr) Addr() string {
	return g.addr
}

func (g *grpcsvr) Type() string {
	return "gRPC"
}
