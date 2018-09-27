package codegen

import (
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ClientTypeFiles returns the gRPC transport type files.
func ClientTypeFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	fw := make([]*codegen.File, len(root.API.GRPC.Services))
	seen := make(map[string]struct{})
	for i, r := range root.API.GRPC.Services {
		fw[i] = clientType(genpkg, r, seen)
	}
	return fw
}

// clientType returns the file containing the constructor functions to
// transform the service payload types to the corresponding gRPC request types
// and gRPC response types to the corresponding service result types.
//
// seen keeps track of the constructor names that have already been generated
// to prevent duplicate code generation.
func clientType(genpkg string, svc *expr.GRPCServiceExpr, seen map[string]struct{}) *codegen.File {
	var (
		path     string
		initData []*InitData

		sd = GRPCServices.Get(svc.Name())
	)
	{
		path = filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "client", "types.go")
		for _, a := range svc.GRPCEndpoints {
			ed := sd.Endpoint(a.Name())
			if c := ed.Request.ClientConvert; c != nil && c.Init != nil {
				initData = append(initData, c.Init)
			}
			if c := ed.Response.ClientConvert; c != nil && c.Init != nil {
				initData = append(initData, c.Init)
			}
			if ed.ClientStream != nil {
				if c := ed.ClientStream.RecvConvert; c != nil && c.Init != nil {
					initData = append(initData, c.Init)
				}
				if c := ed.ClientStream.SendConvert; c != nil && c.Init != nil {
					initData = append(initData, c.Init)
				}
			}
		}
	}

	header := codegen.Header(svc.Name()+" gRPC client types", "client",
		[]*codegen.ImportSpec{
			{Path: "unicode/utf8"},
			{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: sd.Service.PkgName},
			{Path: "goa.design/goa", Name: "goa"},
			{Path: genpkg + "/grpc/" + codegen.SnakeCase(svc.Name()), Name: svc.Name() + "pb"},
		},
	)
	sections := []*codegen.SectionTemplate{header}
	for _, init := range initData {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "client-type-init",
			Source: typeInitT,
			Data:   init,
		})
	}

	return &codegen.File{Path: path, SectionTemplates: sections}
}
