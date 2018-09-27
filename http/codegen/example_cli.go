package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleCLI returns an example client tool implementation.
func ExampleCLI(genpkg string, root *expr.RootExpr) *codegen.File {
	path := filepath.Join("cmd", codegen.SnakeCase(codegen.Goify(root.API.Name, true))+"_cli", "http.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	idx := strings.LastIndex(genpkg, string(os.PathSeparator))
	rootPath := "."
	if idx > 0 {
		rootPath = genpkg[:idx]
	}
	apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json"},
		{Path: "fmt"},
		{Path: "flag"},
		{Path: "net/url"},
		{Path: "net/http"},
		{Path: "os"},
		{Path: "time"},
		{Path: "github.com/gorilla/websocket"},
		{Path: "goa.design/goa"},
		{Path: "goa.design/goa/http", Name: "goahttp"},
		{Path: rootPath, Name: apiPkg},
		{Path: genpkg + "/http/cli"},
	}
	svcdata := make([]*ServiceData, 0, len(root.API.HTTP.Services))
	for _, svc := range root.API.HTTP.Services {
		svcdata = append(svcdata, HTTPServices.Get(svc.Name()))
	}
	data := map[string]interface{}{
		"Services": svcdata,
		"APIPkg":   apiPkg,
		"APIName":  root.API.Name,
	}
	sections := []*codegen.SectionTemplate{
		codegen.Header("", "main", specs),
		&codegen.SectionTemplate{
			Name:   "do-http-cli",
			Source: doHTTPT,
			Data:   data,
			FuncMap: map[string]interface{}{
				"needStreaming": needStreaming,
			},
		},
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// needStreaming returns true if at least one endpoint in the service
// uses stream for sending payload/result.
func needStreaming(data []*ServiceData) bool {
	for _, s := range data {
		if streamingEndpointExists(s) {
			return true
		}
	}
	return false
}

// input: map[string]interface{}{"Services":[]ServiceData, "APIPkg": string, "APIName": string}
const doHTTPT = `func httpDo(addr string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
	var (
		scheme string
		host string
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

	{{ if needStreaming .Services }}
	var (
    dialer *websocket.Dialer
		connConfigFn goahttp.ConnConfigureFunc
  )
  {
    dialer = websocket.DefaultDialer
  }
	{{ end }}

	return cli.ParseEndpoint(
		scheme,
		host,
		doer,
		goahttp.RequestEncoder,
		goahttp.ResponseDecoder,
		debug,
		{{- if needStreaming .Services }}
		dialer,
		connConfigFn,
		{{- end }}
		{{- range .Services }}
			{{- range .Endpoints }}
			  {{- if .MultipartRequestDecoder }}
		{{ $.APIPkg }}.{{ .MultipartRequestEncoder.FuncName }},
				{{- end }}
			{{- end }}
		{{- end }}
	)
}

func httpUsageCommands() string {
  return cli.UsageCommands()
}

func httpUsageExamples() string {
  return cli.UsageExamples()
}
`
