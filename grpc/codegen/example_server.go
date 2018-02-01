package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	grpcdesign "goa.design/goa/grpc/design"
)

// ExampleServerFiles returns and example main and dummy service
// implementations.
func ExampleServerFiles(genpkg string, root *grpcdesign.RootExpr) *codegen.File {
	return exampleServer(genpkg, root)
}

func exampleServer(genpkg string, root *grpcdesign.RootExpr) *codegen.File {
	var (
		mainPath string
		apiPkg   string
	)
	{
		apiPkg = strings.ToLower(codegen.Goify(root.Design.API.Name, false))
		mainPath = filepath.Join("cmd", codegen.SnakeCase(codegen.Goify(root.Design.API.Name, true))+"_svc", "grpc.go")
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
			{Path: "google.golang.org/grpc"},
			{Path: "goa.design/goa/grpc", Name: "goagrpc"},
			{Path: rootPath, Name: apiPkg},
		}
		for _, svc := range root.GRPCServices {
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
		svcdata := make([]*ServiceData, 0, len(root.GRPCServices))
		for _, svc := range root.GRPCServices {
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
		})
	}
	return &codegen.File{Path: mainPath, SectionTemplates: sections}
}

// input: map[string]interface{}{"Services":[]ServiceData, "APIPkg": string}
const serveGRPCT = `func grpcServe(addr string{{ range .Services }}{{ if .Endpoints }}, {{ .Service.VarName }}Endpoints *{{ .Service.PkgName }}.Endpoints{{ end }}{{ end }}, errc chan error, logger *log.Logger, debug bool) *grpc.Server {
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

	// Initialize gRPC server using default configuration.
	srv := grpc.NewServer()

	// Register the servers.
	{{- range .Services }}
	{{ .PkgName }}.RegisterCalcServer(srv, {{ .Service.VarName }}Server)
	{{- end }}

	// Start gRPC server using default configuration, change the code to
	// configure the server as required by your service.
	go func() {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
      logger.Fatalf("failed to listen: %v", err)
      errc <- err
    }
    logger.Printf("gRPC listening on %s", addr)
		errc <- srv.Serve(lis)
	}()

	return srv
}

func grpcStop(srv *grpc.Server) {
	srv.Stop()
}
`
