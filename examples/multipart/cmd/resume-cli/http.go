package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"goa.design/goa"
	resume "goa.design/goa/examples/multipart"
	cli "goa.design/goa/examples/multipart/gen/http/cli/resume"
	goahttp "goa.design/goa/http"
)

func doHTTP(ctx context.Context, scheme, host string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
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

	return cli.ParseEndpoint(
		scheme,
		host,
		doer,
		goahttp.RequestEncoder,
		goahttp.ResponseDecoder,
		debug,
		resume.ResumeAddEncoderFunc,
	)
}
func httpUsageCommands() string {
	return cli.UsageCommands()
}

func httpUsageExamples() string {
	return cli.UsageExamples()
}
