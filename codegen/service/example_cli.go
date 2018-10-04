package service

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleCLI returns an example client tool main implementation.
func ExampleCLI(genpkg string, root *expr.RootExpr) []*codegen.File {
	var fw []*codegen.File
	for _, svr := range root.API.Servers {
		if m := exampleCLIMain(genpkg, root, svr); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleCLIMain(genpkg string, root *expr.RootExpr, svr *expr.ServerExpr) *codegen.File {
	pkg := codegen.SnakeCase(codegen.Goify(svr.Name, true))
	path := filepath.Join("cmd", pkg+"-cli", "main.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "strings"},
		{Path: "goa.design/goa"},
	}
	data := map[string]interface{}{
		"APIName":    root.API.Name,
		"Server":     codegen.Servers.Get(svr),
		"Transports": getTransports(root),
	}
	sections := []*codegen.SectionTemplate{
		codegen.Header("", "main", specs),
		&codegen.SectionTemplate{
			Name:   "cli-main",
			Source: clientMainT,
			Data:   data,
			FuncMap: map[string]interface{}{
				"join":             strings.Join,
				"toUpper":          strings.ToUpper,
				"defaultTransport": defaultTransport,
			},
		},
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// defaultTransport returns the default transport. If multiple transports
// are defined, it returns the transport data corresponding to HTTP transport.
func defaultTransport(transports []*transportData) *transportData {
	if len(transports) == 1 {
		return transports[0]
	}
	for _, t := range transports {
		if t.Type == codegen.TransportHTTP {
			return t
		}
	}
	return nil // bug
}

// input: map[string]interface{}{"APIName": string, "Server": *codegen.ServerData, "Transports": []*transportData}
const clientMainT = `func main() {
{{- $defaultTransport := (defaultTransport .Transports) }}
  var (
		hostF = flag.String("host", "{{ .Server.DefaultHost.Name }}", "Server host (valid values: {{ (join .Server.AvailableHosts ", ") }})")
		addrF = flag.String("url", "", "` + "`" + `URL` + "`" + ` to service host")
	{{ range .Server.Variables }}
		{{ .VarName }}F = flag.String("{{ .Name }}", {{ printf "%q" .DefaultValue }}, {{ printf "%q" .Description }})
	{{- end }}
		verboseF = flag.Bool("verbose", false, "Print request and response details")
		vF = flag.Bool("v", false, "Print request and response details")
		timeoutF = flag.Int("timeout", 30, "Maximum number of ` + "`" + `seconds` + "`" + ` to wait for response")
  )
  flag.Usage = usage
  flag.Parse()

	var (
		addr string
		timeout int
		debug bool
	)
	{
		addr = *addrF
		if addr == "" {
			switch *hostF {
		{{- range $h := .Server.Hosts }}
			case {{ printf "%q" $h.Name }}:
				addr = {{ printf "%q" ($h.URL $defaultTransport.Type) }}
			{{- range $h.Variables }}
				addr = strings.Replace(addr, {{ printf "\"{%s}\"" .Name }}, *{{ .VarName }}F, -1)
			{{- end }}
		{{- end }}
			default:
				fmt.Fprintln(os.Stderr, "invalid host argument: %q (valid hosts: {{ join .Server.AvailableHosts "|" }}", *hostF)
			}
		}
		timeout = *timeoutF
		debug = *verboseF || *vF
	}

	var (
		scheme string
		host string
	)
	{
		u, err := url.Parse(addr)
    if err != nil {
      fmt.Fprintln(os.Stderr, "invalid URL %#v: %s", addr, err)
      os.Exit(1)
    }
    scheme = u.Scheme
    host = u.Host
	}

	var(
		endpoint goa.Endpoint
		payload interface{}
		err error
	)
	{
		switch scheme {
	{{- range $t := .Transports }}
		case "{{ $t.Type }}", "{{ $t.Type }}s":
			endpoint, payload, err = do{{ toUpper $t.Name }}(scheme, host, timeout, debug)
	{{- end }}
	default:
		fmt.Fprintln(os.Stderr, "invalid scheme: %q (valid schemes: {{ join .Server.Schemes "|" }})", scheme)
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
		m, _ := json.MarshalIndent(data, "", "    ")
		fmt.Println(string(m))
	}
}

func usage() {
  fmt.Fprintf(os.Stderr, ` + "`" + `%s is a command line client for the {{ .APIName }} API.

Usage:
    %s [-url URL][-timeout SECONDS][-verbose|-v] SERVICE ENDPOINT [flags]

    -url URL:    specify service URL (http://localhost:8080)
    -timeout:    maximum number of seconds to wait for response (30)
    -verbose|-v: print request and response details (false)

Commands:
%s
Additional help:
    %s SERVICE [ENDPOINT] --help

Example:
%s
` + "`" + `, os.Args[0], os.Args[0], indent({{ $defaultTransport.Type }}UsageCommands()), os.Args[0], indent({{ $defaultTransport.Type }}UsageExamples()))
}

func indent(s string) string {
  if s == "" {
    return ""
  }
  return "    " + strings.Replace(s, "\n", "\n    ", -1)
}
`
