package codegen

import (
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ServerTypeFiles returns the gRPC transport type files.
func ServerTypeFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	fw := make([]*codegen.File, len(root.API.GRPC.Services))
	seen := make(map[string]struct{})
	for i, r := range root.API.GRPC.Services {
		fw[i] = serverType(genpkg, r, seen)
	}
	return fw
}

// serverType returns the file containing the constructor functions to
// transform the gRPC request types to the corresponding service payload types
// and service result types to the corresponding gRPC response types.
//
// seen keeps track of the constructor names that have already been generated
// to prevent duplicate code generation.
func serverType(genpkg string, svc *expr.GRPCServiceExpr, seen map[string]struct{}) *codegen.File {
	var (
		path     string
		initData []*InitData

		sd = GRPCServices.Get(svc.Name())
	)
	{
		path = filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "server", "types.go")
		for _, a := range svc.GRPCEndpoints {
			ed := sd.Endpoint(a.Name())
			if c := ed.Request.ServerConvert; c != nil && c.Init != nil {
				initData = append(initData, c.Init)
			}
			if c := ed.Response.ServerConvert; c != nil && c.Init != nil {
				initData = append(initData, c.Init)
			}
			if ed.ServerStream != nil {
				if c := ed.ServerStream.SendConvert; c != nil && c.Init != nil {
					initData = append(initData, c.Init)
				}
				if c := ed.ServerStream.RecvConvert; c != nil && c.Init != nil {
					initData = append(initData, c.Init)
				}
			}
		}
	}

	header := codegen.Header(svc.Name()+" gRPC server types", "server",
		[]*codegen.ImportSpec{
			{Path: "unicode/utf8"},
			{Path: "goa.design/goa", Name: "goa"},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name())), Name: sd.Service.PkgName},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name()), "views"), Name: sd.Service.ViewsPkg},
			{Path: filepath.Join(genpkg, "grpc", codegen.SnakeCase(svc.Name())), Name: svc.Name() + "pb"},
		},
	)
	sections := []*codegen.SectionTemplate{header}
	for _, init := range initData {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "server-type-init",
			Source: typeInitT,
			Data:   init,
		})
	}

	return &codegen.File{Path: path, SectionTemplates: sections}
}

// input: InitData
const typeInitT = `{{ comment .Description }}
func {{ .Name }}({{ range .Args }}{{ .Name }} {{ .TypeRef }}, {{ end }}) {{ .ReturnTypeRef }} {
  {{ .Code }}
{{- if .ReturnIsStruct }}
	{{- range .Args }}
		{{- if .FieldName }}
			{{ $.ReturnVarName }}.{{ .FieldName }} = {{ if .Pointer }}&{{ end }}{{ .Name }}
		{{- end }}
	{{- end }}
{{- end }}
  return {{ .ReturnVarName }}
}
`
