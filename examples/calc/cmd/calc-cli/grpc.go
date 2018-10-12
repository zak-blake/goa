package main

import (
	"fmt"
	"os"

	"goa.design/goa"
	"goa.design/goa/examples/calc/gen/grpc/cli"
	"google.golang.org/grpc"
)

func doGRPC(scheme, host string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("could not connect to GRPC server at %s: %v", host, err))
	}
	return cli.ParseEndpoint(conn, nil, nil)
}

func grpcUsageCommands() string {
	return cli.UsageCommands()
}

func grpcUsageExamples() string {
	return cli.UsageExamples()
}
