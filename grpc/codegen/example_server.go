package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleServerFiles returns and example main and dummy service
// implementations.
func ExampleServerFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	var fw []*codegen.File
	for _, svr := range root.API.Servers {
		if m := exampleServer(genpkg, root, svr); m != nil {
			fw = append(fw, m)
		}
	}
	return fw
}

func exampleServer(genpkg string, root *expr.RootExpr, svr *expr.ServerExpr) *codegen.File {
	var (
		mainPath string
		apiPkg   string
	)
	{
		apiPkg = strings.ToLower(codegen.Goify(root.API.Name, false))
		pkg := codegen.SnakeCase(codegen.Goify(svr.Name, true))
		mainPath = filepath.Join("cmd", pkg, "grpc.go")
		if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
			return nil // file already exists, skip it.
		}
	}
	var (
		specs []*codegen.ImportSpec
	)
	{
		idx := strings.LastIndex(genpkg, string(os.PathSeparator))
		rootPath := "."
		if idx > 0 {
			rootPath = genpkg[:idx]
		}
		specs = []*codegen.ImportSpec{
			{Path: "log"},
			{Path: "net"},
			{Path: "os"},
			{Path: "goa.design/goa/grpc/middleware"},
			{Path: "google.golang.org/grpc"},
			{Path: "github.com/grpc-ecosystem/go-grpc-middleware", Name: "grpcmiddleware"},
			{Path: "goa.design/goa/grpc", Name: "goagrpc"},
			{Path: rootPath, Name: apiPkg},
		}
		for _, svc := range root.API.GRPC.Services {
			pkgName := GRPCServices.Get(svc.Name()).Service.PkgName
			specs = append(specs, &codegen.ImportSpec{
				Path: filepath.Join(genpkg, "grpc", codegen.SnakeCase(svc.Name()), "server"),
				Name: pkgName + "svr",
			})
			specs = append(specs, &codegen.ImportSpec{
				Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name())),
				Name: pkgName,
			})
			specs = append(specs, &codegen.ImportSpec{
				Path: filepath.Join(genpkg, "grpc", codegen.SnakeCase(svc.Name())),
				Name: svc.Name() + "pb",
			})
		}
	}
	var (
		sections []*codegen.SectionTemplate
	)
	{
		sections = []*codegen.SectionTemplate{codegen.Header("", "main", specs)}
		svcdata := make([]*ServiceData, 0, len(root.API.GRPC.Services))
		for _, svc := range root.API.GRPC.Services {
			svcdata = append(svcdata, GRPCServices.Get(svc.Name()))
		}
		data := map[string]interface{}{
			"Services": svcdata,
			"APIPkg":   apiPkg,
		}
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "serve-grpc",
			Source: serveGRPCT,
			Data:   data,
			FuncMap: map[string]interface{}{
				"goify": codegen.Goify,
			},
		})
	}
	return &codegen.File{Path: mainPath, SectionTemplates: sections, SkipExist: true}
}

// input: map[string]interface{}{"Services":[]ServiceData, "APIPkg": string}
const serveGRPCT = `{{ comment "grpcsvr implements Server interface." }}
type grpcsvr struct {
	svr *grpc.Server
	addr string
}

func newGRPCServer(scheme, host string{{ range .Services }}{{ if .Endpoints }}, {{ .Service.VarName }}Endpoints *{{ .Service.PkgName }}.Endpoints{{ end }}{{ end }}, logger *log.Logger, debug bool) Server {
	// Setup goa log adapter. Replace logger with your own using your
  // log package of choice. The goa.design/middleware/logging/...
  // packages define log adapters for common log packages.
  var (
    adapter middleware.Logger
  )
  {
    adapter = middleware.NewLogger(logger)
  }

	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to gRPC requests and
	// responses.
	var (
	{{- range .Services }}
		{{ .Service.VarName }}Server *{{.Service.PkgName}}svr.Server
	{{- end }}
	)
	{
	{{- range .Services }}
		{{- if .Endpoints }}
		{{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New({{ .Service.VarName }}Endpoints)
		{{-  else }}
		{{ .Service.VarName }}Server = {{ .Service.PkgName }}svr.New(nil)
		{{-  end }}
	{{- end }}
	}

	// Initialize gRPC server with the middleware.
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			middleware.RequestID(),
      middleware.Log(adapter),
	  )),
	)

	// Register the servers.
	{{- range .Services }}
	{{ .PkgName }}.Register{{ goify .Service.VarName true }}Server(srv, {{ .Service.VarName }}Server)
	{{- end }}

	return &grpcsvr{svr: srv, addr: host}
}

func (g *grpcsvr) Start(errc chan error) {
	go func() {
		lis, err := net.Listen("tcp", g.addr)
		if err != nil {
      errc <- err
    }
		errc <- g.svr.Serve(lis)
	}()
}

func (g *grpcsvr) Stop() error {
	g.svr.Stop()
	return nil
}

func (g *grpcsvr) Addr() string {
	return g.addr
}

func (g *grpcsvr) Type() string {
  return "gRPC"
}
`
