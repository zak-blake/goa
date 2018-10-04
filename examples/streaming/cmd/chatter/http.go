package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	chattersvc "goa.design/goa/examples/streaming/gen/chatter"
	chattersvcsvr "goa.design/goa/examples/streaming/gen/http/chatter/server"
	goahttp "goa.design/goa/http"
	"goa.design/goa/http/middleware"
)

// httpsvr implements Server interface.
type httpsvr struct {
	svr  *http.Server
	addr string
}

func newHTTPServer(scheme, host string, chatterEndpoints *chattersvc.Endpoints, logger *log.Logger, debug bool) Server {
	// Setup logger and goa log adapter. Replace logger with your own using
	// your log package of choice. The goa.design/middleware/logging/...
	// packages define log adapters for common log packages.
	var (
		adapter middleware.Logger
	)
	{
		adapter = middleware.NewLogger(logger)
	}

	// Provide the transport specific request decoder and response encoder.
	// The goa http package has built-in support for JSON, XML and gob.
	// Other encodings can be used by providing the corresponding functions,
	// see goa.design/encoding.
	var (
		dec = goahttp.RequestDecoder
		enc = goahttp.ResponseEncoder
	)

	// Build the service HTTP request multiplexer and configure it to serve
	// HTTP requests to the service endpoints.
	var mux goahttp.Muxer
	{
		mux = goahttp.NewMuxer()
	}

	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to HTTP requests and
	// responses.
	var (
		chatterServer *chattersvcsvr.Server
	)
	{
		eh := errorHandler(logger)
		upgrader := &websocket.Upgrader{}
		chatterServer = chattersvcsvr.New(chatterEndpoints, mux, dec, enc, eh, upgrader, nil)
	}

	// Configure the mux.
	chattersvcsvr.Mount(mux, chatterServer)
	for _, m := range chatterServer.Mounts {
		logger.Printf("method %q mounted on %s %s", m.Method, m.Verb, m.Pattern)
	}

	// Wrap the multiplexer with additional middlewares. Middlewares mounted
	// here apply to all the service endpoints.
	var handler http.Handler = mux
	{
		if debug {
			handler = middleware.Debug(mux, os.Stdout)(handler)
		}
		handler = middleware.Log(adapter)(handler)
		handler = middleware.RequestID()(handler)
	}

	// Start HTTP server using default configuration, change the code to
	// configure the server as required by your service.
	srv := &http.Server{Addr: host, Handler: handler}

	return &httpsvr{svr: srv, addr: host}
}

func (h *httpsvr) Start(errc chan error) {
	go func() {
		errc <- h.svr.ListenAndServe()
	}()
}

func (h *httpsvr) Stop() error {
	// Shutdown gracefully with a 30s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return h.svr.Shutdown(ctx)
}

func (h *httpsvr) Addr() string {
	return h.addr
}

func (h *httpsvr) Type() string {
	return "HTTP"
}

// errorHandler returns a function that writes and logs the given error.
// The function also writes and logs the error unique ID so that it's possible
// to correlate.
func errorHandler(logger *log.Logger) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, w http.ResponseWriter, err error) {
		id := ctx.Value(middleware.RequestIDKey).(string)
		w.Write([]byte("[" + id + "] encoding: " + err.Error()))
		logger.Printf("[%s] ERROR: %s", id, err.Error())
	}
}
