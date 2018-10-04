package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleCLI returns an example client tool main implementation.
func ExampleCLI(genpkg string, root *expr.RootExpr) []*codegen.File {
	var files []*codegen.File
	for _, svr := range root.API.Servers {
		pkg := codegen.SnakeCase(codegen.Goify(svr.Name, true))
		apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
		path := filepath.Join("cmd", pkg+"-cli", "http.go")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			continue // file already exists, skip it.
		}
		idx := strings.LastIndex(genpkg, string(os.PathSeparator))
		rootPath := "."
		if idx > 0 {
			rootPath = genpkg[:idx]
		}
		specs := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "encoding/json"},
			{Path: "flag"},
			{Path: "fmt"},
			{Path: "net/http"},
			{Path: "net/url"},
			{Path: "os"},
			{Path: "strings"},
			{Path: "time"},
			{Path: "github.com/gorilla/websocket"},
			{Path: "goa.design/goa"},
			{Path: "goa.design/goa/http", Name: "goahttp"},
			{Path: genpkg + "/http/cli/" + pkg, Name: "cli"},
			{Path: rootPath, Name: apiPkg},
		}
		svcdata := make([]*ServiceData, len(svr.Services))
		for i, svc := range svr.Services {
			svcdata[i] = HTTPServices.Get(svc)
		}
		data := map[string]interface{}{
			"Services": svcdata,
			"APIPkg":   apiPkg,
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
		files = append(files, &codegen.File{
			Path:             path,
			SectionTemplates: sections,
			SkipExist:        true,
		})
	}
	return files
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

// input: map[string]interface{}{"Services":[]ServiceData, "APIPkg": string, "ServerName": string}
const doHTTPT = `func doHTTP(scheme, host string, timeout int, debug bool) (goa.Endpoint, interface{}, error) {
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
