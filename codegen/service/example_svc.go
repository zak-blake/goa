package service

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

type (
	// dummyEndpointData contains the data needed to render dummy endpoint
	// implementation in the dummy service file.
	dummyEndpointData struct {
		*MethodData
		// ServiceVarName is the service variable name.
		ServiceVarName string
		// PayloadFullRef is the fully qualified reference to the payload.
		PayloadFullRef string
		// ResultFullName is the fully qualified name of the result.
		ResultFullName string
		// ResultFullRef is the fully qualified reference to the result.
		ResultFullRef string
		// ResultIsStruct indicates that the result type is a struct.
		ResultIsStruct bool
		// ResultView is the view to render the result. It is set only if the
		// result type uses views.
		ResultView string
	}

	// transportData contains the data about a transport (http or grpc).
	transportData struct {
		// Type is the transport type.
		Type codegen.Transport
		// Name is the transport name.
		Name string
	}
)

// ExampleServiceFiles returns a dummy service implementation and
// example service main.go.
func ExampleServiceFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	var fw []*codegen.File
	for _, svc := range root.Services {
		if f := dummyServiceFile(genpkg, root, svc); f != nil {
			fw = append(fw, f)
		}
	}
	for _, svr := range root.API.Servers {
		if m := exampleSvrMain(genpkg, root, svr); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

// dummyServiceFile returns a dummy implementation of the given service.
func dummyServiceFile(genpkg string, root *expr.RootExpr, svc *expr.ServiceExpr) *codegen.File {
	path := codegen.SnakeCase(svc.Name) + ".go"
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	data := Services.Get(svc.Name)
	apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
	sections := []*codegen.SectionTemplate{
		codegen.Header("", apiPkg, []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "log"},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name)), Name: data.PkgName},
		}),
		{
			Name:   "dummy-service",
			Source: dummyServiceStructT,
			Data:   data,
		},
	}
	for _, m := range svc.Methods {
		md := data.Method(m.Name)
		ed := &dummyEndpointData{
			MethodData:     md,
			ServiceVarName: data.VarName,
		}
		if m.Payload.Type != expr.Empty {
			ed.PayloadFullRef = data.Scope.GoFullTypeRef(m.Payload, data.PkgName)
		}
		if m.Result.Type != expr.Empty {
			ed.ResultFullName = data.Scope.GoFullTypeName(m.Result, data.PkgName)
			ed.ResultFullRef = data.Scope.GoFullTypeRef(m.Result, data.PkgName)
			ed.ResultIsStruct = expr.IsObject(m.Result.Type)
			if md.ViewedResult != nil {
				view := "default"
				if m.Result.Meta != nil {
					if v, ok := m.Result.Meta["view"]; ok {
						view = v[0]
					}
				}
				ed.ResultView = view
			}
		}
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "dummy-endpoint",
			Source: dummyEndpointImplT,
			Data:   ed,
		})
	}

	return &codegen.File{
		Path:             path,
		SectionTemplates: sections,
		SkipExist:        true,
	}
}

func exampleSvrMain(genpkg string, root *expr.RootExpr, svr *expr.ServerExpr) *codegen.File {
	pkg := codegen.SnakeCase(codegen.Goify(svr.Name, true))
	mainPath := filepath.Join("cmd", pkg, "main.go")
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	idx := strings.LastIndex(genpkg, string(os.PathSeparator))
	rootPath := "."
	if idx > 0 {
		rootPath = genpkg[:idx]
	}
	apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
	specs := []*codegen.ImportSpec{
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "log"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "os/signal"},
		{Path: "strings"},
		{Path: rootPath, Name: apiPkg},
	}
	svcdata := make([]*Data, 0, len(root.Services))
	for _, svc := range root.Services {
		sd := Services.Get(svc.Name)
		svcdata = append(svcdata, sd)
		specs = append(specs, &codegen.ImportSpec{
			Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name)),
			Name: sd.PkgName,
		})
	}
	data := map[string]interface{}{
		"Services":   svcdata,
		"APIPkg":     apiPkg,
		"Server":     codegen.Servers.Get(svr),
		"Transports": getTransports(root),
	}
	sections := []*codegen.SectionTemplate{codegen.Header("", "main", specs)}
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "service-main",
		Source: mainT,
		Data:   data,
		FuncMap: map[string]interface{}{
			"toUpper":      strings.ToUpper,
			"join":         strings.Join,
			"transportFor": transportFor,
		},
	})

	return &codegen.File{Path: mainPath, SectionTemplates: sections, SkipExist: true}
}

func getTransports(root *expr.RootExpr) []*transportData {
	var transports []*transportData
	seen := make(map[codegen.Transport]struct{})
	for _, svc := range root.Services {
		if _, ok := seen[codegen.TransportHTTP]; !ok {
			if root.API.HTTP.Service(svc.Name) != nil {
				transports = append(transports, &transportData{Type: codegen.TransportHTTP, Name: "HTTP"})
				seen[codegen.TransportHTTP] = struct{}{}
			}
		}
		if _, ok := seen[codegen.TransportGRPC]; !ok {
			if root.API.GRPC.Service(svc.Name) != nil {
				transports = append(transports, &transportData{Type: codegen.TransportGRPC, Name: "gRPC"})
				seen[codegen.TransportGRPC] = struct{}{}
			}
		}
	}
	return transports
}

// transportFor returns the transport data for the given transport type.
func transportFor(transports []*transportData, t codegen.Transport) *transportData {
	for _, tr := range transports {
		if tr.Type == t {
			return tr
		}
	}
	return nil
}

// input: Data
const dummyServiceStructT = `{{ printf "%s service example implementation.\nThe example methods log the requests and return zero values." .Name | comment }}
type {{ .VarName }}srvc struct {
	logger *log.Logger
}

{{ printf "New%s returns the %s service implementation." .StructName .Name | comment }}
func New{{ .StructName }}(logger *log.Logger) {{ .PkgName }}.Service {
	return &{{ .VarName }}srvc{logger}
}
`

// input: endpointData
const dummyEndpointImplT = `{{ comment .Description }}
{{- if .ServerStream }}
func (s *{{ .ServiceVarName }}srvc) {{ .VarName }}(ctx context.Context{{ if .PayloadFullRef }}, p {{ .PayloadFullRef }}{{ end }}, stream {{ .ServerStream.Interface }}) (err error) {
{{- else }}
func (s *{{ .ServiceVarName }}srvc) {{ .VarName }}(ctx context.Context{{ if .PayloadFullRef }}, p {{ .PayloadFullRef }}{{ end }}) ({{ if .ResultFullRef }}res {{ .ResultFullRef }}, {{ if .ViewedResult }}{{ if not .ViewedResult.ViewName }}view string, {{ end }}{{ end }} {{ end }}err error) {
{{- end }}
{{- if and (and .ResultFullRef .ResultIsStruct) (not .ServerStream) }}
	res = &{{ .ResultFullName }}{}
{{- end }}
{{- if .ResultView }}
	{{- if .ServerStream }}
	stream.SetView({{ printf "%q" .Result.View }})
	{{- else }}
	view = {{ printf "%q" .ResultView }}
	{{- end }}
{{- end }}
	s.logger.Print("{{ .ServiceVarName }}.{{ .Name }}")
	return
}
`

// input: map[string]interface{}{"Services": []ServiceData, "APIPkg": string, "Server": *codegen.ServerData, "Transports": []*transportData}
const mainT = `// Server provides the means to start and stop a server.
type Server interface {
	{{ comment "Start starts a server and sends any errors to the error channel." }}
  Start(errc chan error)
	{{ comment "Stop stops a server." }}
  Stop() error
	{{ comment "Addr returns the listen address." }}
  Addr() string
	{{ comment "Type returns the server type (HTTP or gRPC)" }}
	Type() string
}

func main() {
  // Define command line flags, add any other flag required to configure
  // the service.
  var (
		hostF = flag.String("host", "{{ .Server.DefaultHost.Name }}", "Server host (valid values: {{ (join .Server.AvailableHosts ", ") }})")
		domainF = flag.String("domain", "", "Host domain name (overrides host domain and port specified in design)")
{{- range .Transports }}
	{{ .Type }}PortF = flag.String("{{ .Type }}-port", "", "{{ .Name }} port (used in conjunction with -- domain flag)")
{{- end }}
{{- range .Server.Variables }}
		{{ .VarName }}F = flag.String("{{ .Name }}", {{ printf "%q" .DefaultValue }}, {{ printf "%q" .Description }})
{{- end }}
		secureF = flag.Bool("secure", false, "Use secure scheme (https or grpcs)")
    dbgF  = flag.Bool("debug", false, "Log request and response bodies")
  )
  flag.Parse()

	var (
{{- range .Transports }}
		{{ .Type }}Addr string
{{- end }}
	)
	{
		if *domainF != "" {
{{- range .Transports }}
			{{ .Type }}Scheme := "{{ .Type }}"
			if *secureF {
				{{ .Type }}Scheme = "{{ .Type }}s"
			}
			{{ .Type }}Port := *{{ .Type }}PortF
			if {{ .Type }}Port == "" {
				{{ .Type }}Port = {{ if eq .Type "http" }}"80"{{ else }}"8080"{{ end }}
				if *secureF {
					{{ .Type }}Port = {{ if eq .Type "http" }}"443"{{ else }}"8443"{{ end }}
				}
			}
			{{ .Type }}Addr = {{ .Type }}Scheme + "://" + *domainF + ":" + {{ .Type }}Port
{{- end }}
		}
	}

  // Setup logger and goa log adapter. Replace logger with your own using
  // your log package of choice. The goa.design/middleware/logging/...
  // packages define log adapters for common log packages.
  var (
    logger *log.Logger
  )
  {
    logger = log.New(os.Stderr, "[{{ .APIPkg }}] ", log.Ltime)
  }

	// Create the structs that implement the services.
	var (
	{{- range .Services }}
		{{- if .Methods }}
		{{ .VarName }}Svc {{ .PkgName }}.Service
		{{- end }}
	{{- end }}
	)
	{
	{{- range .Services }}
		{{- if .Methods }}
		{{ .VarName }}Svc = {{ $.APIPkg }}.New{{ .StructName }}(logger)
		{{- end }}
	{{- end }}
	}

	// Wrap the services in endpoints that can be invoked from other
	// services potentially running in different processes.
	var (
	{{- range .Services }}
		{{- if .Methods }}
		{{ .VarName }}Endpoints *{{ .PkgName }}.Endpoints
		{{- end }}
	{{- end }}
	)
	{
	{{- range .Services }}{{ $svc := . }}
		{{- if .Methods }}
		{{ .VarName }}Endpoints = {{ .PkgName }}.NewEndpoints({{ .VarName }}Svc{{ range .Schemes }}, {{ $.APIPkg }}.{{ $svc.StructName }}{{ .Type }}Auth{{ end }})
		{{- end }}
	{{- end }}
	}

	// Create channel used by both the signal handler and server goroutines
	// to notify the main goroutine when to stop the server.
	errc := make(chan error)

	// Setup interrupt handler. This optional step configures the process so
	// that SIGINT and SIGTERM signals cause the service to stop gracefully.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		errc <- fmt.Errorf("%s", <-c)
	}()

	var (
		addr string
		u *url.URL
		svrs []Server
	)
	switch *hostF {
{{- range $h := .Server.Hosts }}
	case {{ printf "%q" $h.Name }}:
	{{- range $u := $h.URIs }}
		{{ $t := (transportFor $.Transports $u.Transport) }}
		{{- if $t }}
			if {{ $t.Type }}Addr != "" {
				addr = {{ $t.Type }}Addr
			} else {
				addr = {{ printf "%q" $u.URL }}
			}
			{{- range $h.Variables }}
				addr = strings.Replace(addr, {{ printf "\"{%s}\"" .Name }}, *{{ .VarName }}F, -1)
			{{- end }}
			u = parseAddr(addr)
			svrs = append(svrs, new{{ toUpper $t.Name }}Server(u.Scheme, u.Host, {{ range $.Services }}{{ .VarName }}Endpoints, {{ end }}logger, *dbgF))
		{{- end }}
	{{- end }}
{{- end }}
	default:
		fmt.Fprintf(os.Stderr, "invalid host argument: %q (valid hosts: {{ join .Server.AvailableHosts "|" }})", *hostF)
		os.Exit(1)
	}

	// Start the servers
	for _, s := range svrs {
		logger.Println("Starting " + s.Type() + " server at " + s.Addr())
		s.Start(errc)
	}

	// Wait for signal.
	logger.Printf("exiting (%v)", <-errc)
	for _, s := range svrs {
		logger.Println("Shutting down " + s.Type() + " server at " + s.Addr())
		s.Stop()
	}
	logger.Println("exited")
}

func parseAddr(addr string) *url.URL {
	u, err := url.Parse(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid URL %#v: %s", addr, err)
		os.Exit(1)
	}
	return u
}
`
