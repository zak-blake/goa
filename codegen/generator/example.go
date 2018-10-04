package generator

import (
	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/eval"
	"goa.design/goa/expr"
	grpccodegen "goa.design/goa/grpc/codegen"
	httpcodegen "goa.design/goa/http/codegen"
)

// Example iterates through the roots and returns files that implement an
// example service and client.
func Example(genpkg string, roots []eval.Root) ([]*codegen.File, error) {
	var (
		files []*codegen.File
	)
	for _, root := range roots {
		r, ok := root.(*expr.RootExpr)
		if !ok {
			continue // could be a plugin root expression
		}

		// Auth
		f := service.AuthFuncsFile(genpkg, r)
		if f != nil {
			files = append(files, f)
		}

		// HTTP
		if len(r.API.HTTP.Services) > 0 {
			svcs := make([]string, 0, len(r.API.HTTP.Services))
			for _, s := range r.API.HTTP.Services {
				svcs = append(svcs, s.Name())
			}
			if svrs := httpcodegen.ExampleServerFiles(genpkg, r); len(svrs) > 0 {
				files = append(files, svrs...)
			}
			if cli := httpcodegen.ExampleCLI(genpkg, r); len(cli) > 0 {
				files = append(files, cli...)
			}
		}

		// GRPC
		if len(r.API.GRPC.Services) > 0 {
			svcs := make([]string, 0, len(r.API.GRPC.Services))
			for _, s := range r.API.GRPC.Services {
				svcs = append(svcs, s.Name())
			}
			if svrs := grpccodegen.ExampleServerFiles(genpkg, r); len(svrs) > 0 {
				files = append(files, svrs...)
			}
			if cli := grpccodegen.ExampleCLI(genpkg, r); len(cli) > 0 {
				files = append(files, cli...)
			}
		}

		// server main
		if fs := service.ExampleServiceFiles(genpkg, r); len(fs) != 0 {
			files = append(files, fs...)
		}

		// client main
		if fs := service.ExampleCLI(genpkg, r); len(fs) != 0 {
			files = append(files, fs...)
		}
	}
	return files, nil
}
