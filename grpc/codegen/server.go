package codegen

import (
	"fmt"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/design"
	grpcdesign "goa.design/goa/grpc/design"
)

// ServerFiles returns all the server gRPC transport files.
func ServerFiles(genpkg string, root *grpcdesign.RootExpr) []*codegen.File {
	fw := make([]*codegen.File, len(root.GRPCServices))
	for i, svc := range root.GRPCServices {
		fw[i] = server(genpkg, svc)
	}
	return fw
}

// server returns the files defining the gRPC server.
func server(genpkg string, svc *grpcdesign.ServiceExpr) *codegen.File {
	path := filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "server", "server.go")
	data := GRPCServices.Get(svc.Name())
	title := fmt.Sprintf("%s GRPC server", svc.Name())
	sections := []*codegen.SectionTemplate{
		codegen.Header(title, "server", []*codegen.ImportSpec{
			{Path: "context"},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name())), Name: data.Service.PkgName},
			{Path: filepath.Join(genpkg, "grpc", codegen.SnakeCase(svc.Name())), Name: svc.Name() + "pb"},
		}),
	}

	sections = append(sections, &codegen.SectionTemplate{Name: "server-struct", Source: serverStructT, Data: data})
	sections = append(sections, &codegen.SectionTemplate{Name: "server-init", Source: serverInitT, Data: data})

	for _, e := range data.Endpoints {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "server-grpc-interface",
			Source: serverGRPCInterfaceT,
			Data:   e,
			FuncMap: map[string]interface{}{
				"typeCast": typeCastField,
			},
		})
	}

	return &codegen.File{Path: path, SectionTemplates: sections}
}

// typeCastField type casts the request/response "field" attribute type as per
// the method payload/result type.
// NOTE: If the method payload/result type is not an  object it is wrapped
// into a "field" attribute in the gRPC request/response message type.
func typeCastField(srcVar string, ed *EndpointData, payload bool) string {
	se := grpcdesign.Root.Service(ed.ServiceName)
	ep := se.Endpoint(ed.Name)
	src := ep.Response.Type
	tgt := ep.MethodExpr.Result.Type
	if payload {
		src = ep.Request.Type
		tgt = ep.MethodExpr.Payload.Type
	}
	src = design.AsObject(src).Attribute("field").Type
	return typeCast(srcVar, src, tgt, false)
}

// input: ServiceData
const serverStructT = `{{ printf "%s implements the %s.%s interface." .ServerStruct .PkgName .ServerInterface | comment }}
type {{ .ServerStruct }} struct {
	endpoints *{{ .Service.PkgName }}.Endpoints
}
`

// input: ServiceData
const serverInitT = `{{ printf "%s instantiates the server struct with the %s service endpoints." .ServerInit .Service.Name | comment }}
func {{ .ServerInit }}(e *{{ .Service.PkgName }}.Endpoints) *{{ .ServerStruct }} {
	return &{{ .ServerStruct }}{e}
}
`

// input: EndpointData
const serverGRPCInterfaceT = `{{ printf "%s implements the %q method in %s.%s interface." .VarName .VarName .PkgName .ServerInterface | comment }}
func (s *{{ .ServerStruct }}) {{ .VarName }}(ctx context.Context, p {{ .ServerRequest.Ref }}) ({{ .ServerResponse.Ref }}, error) {
	{{- if .ServerRequest.Init }}
		payload := {{ .ServerRequest.Init.Name }}({{ range .ServerRequest.Init.Args }}{{ .Name }}{{ end }})
	{{- else }}
		payload := {{ typeCast "p.Field" . true }}
	{{- end }}
	v, err := s.endpoints.{{ .VarName }}(ctx, payload)
	if err != nil {
		return nil, err
	}
	res := v.({{ .ResultRef }})
	resp := {{ .ServerResponse.Init.Name }}({{ range .ServerResponse.Init.Args }}{{ .Name }}{{ end }})
	return resp, nil
}
`
