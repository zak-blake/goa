package main

import (
	"fmt"
	"os"

	"goa.design/goa"
	"goa.design/goa/examples/streaming/gen/grpc/cli"
	"google.golang.org/grpc"
)

func grpcDo(addr string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("could not connect to GRPC server at %s: %v", addr, err))
	}
	return cli.ParseEndpoint(conn)
}

func grpcUsageCommands() string {
	return cli.UsageCommands()
}

func grpcUsageExamples() string {
	return cli.UsageExamples()
}
