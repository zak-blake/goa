package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"goa.design/goa"
	chattersvc "goa.design/goa/examples/streaming/gen/chatter"
)

func main() {
	var (
		addrF      = flag.String("url", "http://localhost:8080", "`URL` to service host")
		verboseF   = flag.Bool("verbose", false, "Print request and response details")
		vF         = flag.Bool("v", false, "Print request and response details")
		timeoutF   = flag.Int("timeout", 30, "Maximum number of `seconds` to wait for response")
		transportF = flag.String("transport", "http", "Transport to use for the request (Allowed values: http, grpc)")
	)
	flag.Usage = usage
	flag.Parse()

	var (
		transport string
		timeout   int
		debug     bool
	)
	{
		transport = *transportF
		timeout = *timeoutF
		debug = *verboseF || *vF
	}

	var (
		endpoint goa.Endpoint
		payload  interface{}
		err      error
	)
	{
		switch transport {
		case "http":
			endpoint, payload, err = httpDo(*addrF, timeout, debug)
		case "grpc":
			endpoint, payload, err = grpcDo(*addrF, timeout, debug)
		default:
			fmt.Fprintf(os.Stderr, "unknown transport %q", transport)
			os.Exit(1)
		}
	}
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr, "run '"+os.Args[0]+" --help' for detailed usage.")
		os.Exit(1)
	}

	data, err := endpoint(context.Background(), payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if data != nil && !debug {
		switch stream := data.(type) {
		case chattersvc.EchoerClientStream:
			// bidirectional streaming
			trapCtrlC(stream)
			fmt.Println("Press Ctrl+D to stop chatting.")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				p := scanner.Text()
				if err := stream.Send(p); err != nil {
					fmt.Println(fmt.Errorf("Error sending into stream: %s", err))
					os.Exit(1)
				}
				d, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println(fmt.Errorf("Error reading from stream: %s", err))
				}
				prettyPrint(d)
			}
			if err := stream.Close(); err != nil {
				fmt.Println(fmt.Errorf("Error closing stream: %s", err))
			}
		case chattersvc.ListenerClientStream:
			// payload streaming (no server response)
			trapCtrlC(stream)
			fmt.Println("Press Ctrl+D to stop chatting.")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				p := scanner.Text()
				if err := stream.Send(p); err != nil {
					fmt.Println(fmt.Errorf("Error sending into stream: %s", err))
					os.Exit(1)
				}
			}
			if err := stream.Close(); err != nil {
				fmt.Println(fmt.Errorf("Error closing stream: %s", err))
			}
		case chattersvc.SummaryClientStream:
			// payload streaming (server responds with a result type)
			trapCtrlC(stream)
			fmt.Println("Press Ctrl+D to stop chatting.")
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				p := scanner.Text()
				if err := stream.Send(p); err != nil {
					fmt.Println(fmt.Errorf("Error sending into stream: %s", err))
					os.Exit(1)
				}
			}
			if p, err := stream.CloseAndRecv(); err != nil {
				fmt.Println(fmt.Errorf("Error closing stream: %s", err))
			} else {
				prettyPrint(p)
			}
		case chattersvc.HistoryClientStream:
			// result streaming
			for {
				p, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println(fmt.Errorf("Error reading from stream: %s", err))
				}
				prettyPrint(p)
			}
		default:
			prettyPrint(data)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `%s is a command line client for the chatter API.

Usage:
    %s [-url URL][-timeout SECONDS][-verbose|-v][-transport NAME] SERVICE ENDPOINT [flags]

    -url URL:    specify service URL (http://localhost:8080)
    -timeout:    maximum number of seconds to wait for response (30)
    -verbose|-v: print request and response details (false)
    -transport:  specify which transport to use (allowed values: http, grpc. Default is http.)

Commands:
%s
Additional help:
    %s SERVICE [ENDPOINT] --help

Example:
%s
`, os.Args[0], os.Args[0], indent(httpUsageCommands()), os.Args[0], indent(httpUsageExamples()))
}

func indent(s string) string {
	if s == "" {
		return ""
	}
	return "    " + strings.Replace(s, "\n", "\n    ", -1)
}

func prettyPrint(s interface{}) {
	m, _ := json.MarshalIndent(s, "", "    ")
	fmt.Println(string(m))
}

// Trap Ctrl+C to gracefully exit the client.
func trapCtrlC(stream interface{}) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func(stream interface{}) {
		for range ch {
			fmt.Println("\nexiting")
			if s, ok := stream.(chattersvc.EchoerClientStream); ok {
				s.Close()
			} else if s, ok := stream.(chattersvc.ListenerClientStream); ok {
				s.Close()
			} else if s, ok := stream.(chattersvc.SummaryClientStream); ok {
				s.CloseAndRecv()
			}
			os.Exit(0)
		}
	}(stream)
}
