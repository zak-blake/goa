package codegen

import (
	"fmt"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ClientFiles returns all the client gRPC transport files.
func ClientFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	fw := make([]*codegen.File, len(root.API.GRPC.Services))
	for i, svc := range root.API.GRPC.Services {
		fw[i] = client(genpkg, svc)
	}
	return fw
}

// client returns the files defining the gRPC client.
func client(genpkg string, svc *expr.GRPCServiceExpr) *codegen.File {
	path := filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "client", "client.go")
	data := GRPCServices.Get(svc.Name())
	title := fmt.Sprintf("%s GRPC client", svc.Name())
	sections := []*codegen.SectionTemplate{
		codegen.Header(title, "client", []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "google.golang.org/grpc"},
			{Path: "goa.design/goa", Name: "goa"},
			{Path: "goa.design/goa/grpc", Name: "goagrpc"},
			{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
			{Path: genpkg + "/grpc/" + codegen.SnakeCase(svc.Name()), Name: svc.Name() + "pb"},
		}),
	}
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "client-struct",
		Source: clientStructT,
		Data:   data,
	})
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "client-init",
		Source: clientInitT,
		Data:   data,
	})
	for _, e := range data.Endpoints {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "client-grpc-interface",
			Source: clientGRPCInterfaceT,
			Data:   e,
			FuncMap: map[string]interface{}{
				"convertType": typeConvertField,
			},
		})
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// input: ServiceData
const clientStructT = `{{ printf "%s lists the service endpoint gRPC clients." .ClientStruct | comment }}
type {{ .ClientStruct }} struct {
	grpccli {{ .PkgName }}.{{ .ClientInterface }}
	opts []grpc.CallOption
}
`

// input: ServiceData
const clientInitT = `{{ printf "New%s instantiates gRPC client for all the %s service servers." .ClientStruct .Service.Name | comment }}
func New{{ .ClientStruct }}(cc *grpc.ClientConn, opts ...grpc.CallOption) *{{ .ClientStruct }} {
  return &{{ .ClientStruct }}{
		grpccli: {{ .ClientInterfaceInit }}(cc),
		opts: opts,
	}
}
`

// input: EndpointData
const clientGRPCInterfaceT = `{{ printf "%s calls the %q function in %s.%s interface." .Method.VarName .Method.VarName .PkgName .ClientInterface | comment }}
func (c *{{ .ClientStruct }}) {{ .Method.VarName }}() goa.Endpoint {
	return func(ctx context.Context, v interface{}) (interface{}, error) {
	{{- if .PayloadRef }}
		p, ok := v.({{ .PayloadRef }})
		if !ok {
			return nil, goagrpc.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .PayloadRef }}", v)
    }
		req := {{ .Request.ClientType.Init.Name }}({{ range .Request.ClientType.Init.Args }}{{ .Name }}, {{ end }})
	{{- end }}
		{{ if .ResultRef }}resp{{ else }}_{{ end }}, err := c.grpccli.{{ .Method.VarName }}(ctx, {{ if .PayloadRef }}req{{ else }}nil{{ end }}, c.opts...)
		if err != nil {
			return nil, err
		}
	{{- if .ResultRef }}
		{{- if .Response.ClientType.Init }}
			res := {{ .Response.ClientType.Init.Name }}({{ range .Response.ClientType.Init.Args }}{{ .Name }}, {{ end }})
		{{- else }}
			res := {{ convertType "resp.Field" . false }}
		{{- end }}
		return res, nil
	{{- else }}
		return nil, nil
	{{- end }}
	}
}
`
