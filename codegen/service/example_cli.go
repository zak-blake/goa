package service

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/design"
)

// ExampleCLI returns an example client tool main implementation.
func ExampleCLI(genpkg string, root *design.RootExpr, transports []*TransportData) *codegen.File {
	path := filepath.Join("cmd", codegen.SnakeCase(codegen.Goify(root.API.Name, true))+"_cli", "main.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	specs := []*codegen.ImportSpec{
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "os"},
		{Path: "strings"},
	}
	data := map[string]interface{}{
		"Transports": transports,
		"APIName":    root.API.Name,
	}
	sections := []*codegen.SectionTemplate{
		codegen.Header("", "main", specs),
		&codegen.SectionTemplate{
			Name:   "cli-main",
			Source: clientMainT,
			Data:   data,
			FuncMap: map[string]interface{}{
				"allowedTransports": allowedTransports,
				"defaultTransport":  defaultTransport,
			},
		},
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// defaultTransport returns the default transport to use in the CLI. If both
// HTTP and gRPC transports are available, it returns "http" as the default
// else it returns the available transport.
func defaultTransport(transports []*TransportData) *TransportData {
	for _, t := range transports {
		if t.IsDefault {
			return t
		}
	}
	panic("no transports found!")
}

// allowedTransports returns the allowed transport names as a string joined by
// comma.
func allowedTransports(transports []*TransportData) string {
	allowed := make([]string, 0, len(transports))
	for _, t := range transports {
		allowed = append(allowed, t.Name())
	}
	return strings.Join(allowed, ", ")
}

// input: map[string]interface{}{"Transports": []*TransportData, "APIName": string}
const clientMainT = `func main() {
	{{- $defaultTransport := defaultTransport .Transports }}
	{{- $allowedTransports := allowedTransports .Transports }}
  var (
    addrF      = flag.String("url", {{ printf "%q" $defaultTransport.URL }}, "` + "`" + `URL` + "`" + ` to service host")
    verboseF   = flag.Bool("verbose", false, "Print request and response details")
    vF         = flag.Bool("v", false, "Print request and response details")
    timeoutF   = flag.Int("timeout", 30, "Maximum number of ` + "`" + `seconds` + "`" + ` to wait for response")
    transportF = flag.String("transport", {{ printf "%q" $defaultTransport.Name }}, "Transport to use for the request (Allowed values: {{ $allowedTransports }})")
  )
  flag.Usage = usage
  flag.Parse()

	var (
		transport string
		timeout int
		debug bool
	)
	{
		transport = *transportF
		timeout = *timeoutF
		debug = *verboseF || *vF
	}

{{- if gt (len .Transports) 1 }}
	switch transport {
		{{- range $t := .Transports }}
	case {{ printf "%q" $t.Name }}:
		{{ $t.Name }}Do(*addrF, timeout, debug)
	{{- end }}
	default:
		fmt.Fprintf(os.Stderr, "invalid transport %#v: %s", transport)
		os.Exit(1)
	}
{{- else }}
	{{ $defaultTransport.Name }}Do(*addrF, timeout, debug)
{{- end }}
}

func usage() {
  fmt.Fprintf(os.Stderr, ` + "`" + `%s is a command line client for the {{ .APIName }} API.

Usage:
    %s [-url URL][-timeout SECONDS][-verbose|-v][-transport NAME] SERVICE ENDPOINT [flags]

    -url URL:    specify service URL (http://localhost:8080)
    -timeout:    maximum number of seconds to wait for response (30)
    -verbose|-v: print request and response details (false)
    -transport:  specify which transport to use (allowed values: {{ $allowedTransports }}. Default is {{ $defaultTransport.Name }}.)

Commands:
%s
Additional help:
    %s SERVICE [ENDPOINT] --help

Example:
%s
` + "`" + `, os.Args[0], os.Args[0], indent({{ $defaultTransport.Name }}UsageCommands()), os.Args[0], indent({{ $defaultTransport.Name }}UsageExamples()))
}

func indent(s string) string {
  if s == "" {
    return ""
  }
  return "    " + strings.Replace(s, "\n", "\n    ", -1)
}
`
