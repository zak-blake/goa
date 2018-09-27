package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"goa.design/goa"
	"goa.design/goa/examples/streaming/gen/http/cli"
	goahttp "goa.design/goa/http"
)

func httpDo(addr string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
	var (
		scheme string
		host   string
	)
	{
		u, err := url.Parse(addr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid URL %#v: %s", addr, err)
			os.Exit(1)
		}
		scheme = u.Scheme
		host = u.Host
		if scheme == "" {
			scheme = "http"
		}
	}

	var (
		doer goahttp.Doer
	)
	{
		doer = &http.Client{Timeout: time.Duration(timeout) * time.Second}
		if debug {
			doer = goahttp.NewDebugDoer(doer)
			doer.(goahttp.DebugDoer).Fprint(os.Stderr)
		}
	}

	var (
		dialer       *websocket.Dialer
		connConfigFn goahttp.ConnConfigureFunc
	)
	{
		dialer = websocket.DefaultDialer
	}

	return cli.ParseEndpoint(
		scheme,
		host,
		doer,
		goahttp.RequestEncoder,
		goahttp.ResponseDecoder,
		debug,
		dialer,
		connConfigFn,
	)
}

func httpUsageCommands() string {
	return cli.UsageCommands()
}

func httpUsageExamples() string {
	return cli.UsageExamples()
}
