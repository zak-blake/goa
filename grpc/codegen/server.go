package codegen

import (
	"fmt"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ServerFiles returns all the server gRPC transport files.
func ServerFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	fw := make([]*codegen.File, len(root.API.GRPC.Services))
	for i, svc := range root.API.GRPC.Services {
		fw[i] = server(genpkg, svc)
	}
	return fw
}

// server returns the files defining the gRPC server.
func server(genpkg string, svc *expr.GRPCServiceExpr) *codegen.File {
	path := filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "server", "server.go")
	data := GRPCServices.Get(svc.Name())
	title := fmt.Sprintf("%s GRPC server", svc.Name())
	sections := []*codegen.SectionTemplate{
		codegen.Header(title, "server", []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "google.golang.org/grpc/codes"},
			{Path: "google.golang.org/grpc/status"},
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
				"convertType": typeConvertField,
			},
		})
	}

	return &codegen.File{Path: path, SectionTemplates: sections}
}

// typeConvertField type converts the request/response "field" attribute type
// as per the method payload/result type.
// NOTE: If the method payload/result type is not an  object it is wrapped
// into a "field" attribute in the gRPC request/response message type.
func typeConvertField(srcVar string, ed *EndpointData, payload bool) string {
	se := expr.Root.API.GRPC.Service(ed.ServiceName)
	ep := se.Endpoint(ed.Method.Name)
	src := ep.Response.Message.Type
	tgt := ep.MethodExpr.Result.Type
	if payload {
		src = ep.Request.Type
		tgt = ep.MethodExpr.Payload.Type
	}
	srcObj := expr.AsObject(src)
	if len(*srcObj) == 0 {
		// empty message type
		return ""
	}
	src = srcObj.Attribute("field").Type
	return typeConvert(srcVar, src, tgt, false)
}

// input: ServiceData
const serverStructT = `{{ printf "%s implements the %s.%s interface." .ServerStruct .PkgName .ServerInterface | comment }}
type {{ .ServerStruct }} struct {
	endpoints *{{ .Service.PkgName }}.Endpoints
}

// ErrorNamer is an interface implemented by generated error structs that
// exposes the name of the error as defined in the expr.
type ErrorNamer interface {
  ErrorName() string
}
`

// input: ServiceData
const serverInitT = `{{ printf "%s instantiates the server struct with the %s service endpoints." .ServerInit .Service.Name | comment }}
func {{ .ServerInit }}(e *{{ .Service.PkgName }}.Endpoints) *{{ .ServerStruct }} {
	return &{{ .ServerStruct }}{e}
}
`

// input: EndpointData
const serverGRPCInterfaceT = `{{ printf "%s implements the %q method in %s.%s interface." .Method.VarName .Method.VarName .PkgName .ServerInterface | comment }}
func (s *{{ .ServerStruct }}) {{ .Method.VarName }}(ctx context.Context, p {{ .Request.ServerType.Ref }}) ({{ .Response.ServerType.Ref }}, error) {
{{- if .PayloadRef }}
	{{- if .Request.ServerType.Init }}
		payload := {{ .Request.ServerType.Init.Name }}({{ range .Request.ServerType.Init.Args }}{{ .Name }}{{ end }})
	{{- else }}
		payload := {{ convertType "p.Field" . true }}
	{{- end }}
{{- end }}
	{{ if .ResultRef }}v{{ else }}_{{ end }}, err := s.endpoints.{{ .Method.VarName }}(ctx, {{ if .PayloadRef }}payload{{ else }}nil{{ end }})
	if err != nil {
	{{- if .Errors }}
		en, ok := err.(ErrorNamer)
		if !ok {
			return nil, err
		}
		switch en.ErrorName() {
		{{- range .Errors }}
		case {{ printf "%q" .Name }}:
			return nil, status.Error({{ .Response.StatusCode }}, err.Error())
		{{- end }}
		}
	{{- else }}
		return nil, err
	{{- end }}
	}
	{{- if .ResultRef }}
		res := v.({{ .ResultRef }})
		resp := {{ .Response.ServerType.Init.Name }}({{ range .Response.ServerType.Init.Args }}{{ .Name }}{{ end }})
		return resp, nil
	{{- else }}
		return nil, nil
	{{- end }}
}
`
