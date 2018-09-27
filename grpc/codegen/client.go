package codegen

import (
	"fmt"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/expr"
)

// ClientFiles returns all the client gRPC transport files.
func ClientFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	svcLen := len(root.API.GRPC.Services)
	fw := make([]*codegen.File, 2*svcLen)
	for i, svc := range root.API.GRPC.Services {
		fw[i] = client(genpkg, svc)
	}
	for i, svc := range root.API.GRPC.Services {
		fw[i+svcLen] = clientEncodeDecode(genpkg, svc)
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
	for _, e := range data.Endpoints {
		if e.ClientStream != nil {
			sections = append(sections, &codegen.SectionTemplate{
				Name:   "client-stream-struct-type",
				Source: streamStructTypeT,
				Data:   e.ClientStream,
			})
		}
	}
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "client-init",
		Source: clientInitT,
		Data:   data,
	})
	for _, e := range data.Endpoints {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "client-endpoint-init",
			Source: clientEndpointInitT,
			Data:   e,
		})
	}
	for _, e := range data.Endpoints {
		if e.ClientStream != nil {
			if e.ClientStream.RecvConvert != nil {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "client-stream-recv",
					Source: streamRecvT,
					Data:   e.ClientStream,
					FuncMap: map[string]interface{}{
						"convertType": typeConvertField,
					},
				})
			}
			if e.Method.StreamKind == expr.ClientStreamKind || e.Method.StreamKind == expr.BidirectionalStreamKind {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "client-stream-send",
					Source: streamSendT,
					Data:   e.ClientStream,
				})
			}
			if e.ServerStream.MustClose {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "client-stream-close",
					Source: streamCloseT,
					Data:   e.ClientStream,
				})
			}
			if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName == "" {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "client-stream-set-view",
					Source: streamSetViewT,
					Data:   e.ClientStream,
				})
			}
		}
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

func clientEncodeDecode(genpkg string, svc *expr.GRPCServiceExpr) *codegen.File {
	var (
		path     string
		sections []*codegen.SectionTemplate

		data = GRPCServices.Get(svc.Name())
	)
	{
		path = filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "client", "encode_decode.go")
		sections = []*codegen.SectionTemplate{
			codegen.Header(svc.Name()+" gRPC client encoders and decoders", "client", []*codegen.ImportSpec{
				{Path: "context"},
				{Path: "strconv"},
				{Path: "google.golang.org/grpc/metadata"},
				{Path: "goa.design/goa", Name: "goa"},
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
				{Path: genpkg + "/grpc/" + codegen.SnakeCase(svc.Name()), Name: svc.Name() + "pb"},
			}),
		}
		fm := transTmplFuncs(svc)
		fm["convertType"] = typeConvertField
		fm["metadataEncodeDecodeData"] = metadataEncodeDecodeData
		for _, e := range data.Endpoints {
			if e.PayloadRef != "" {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "request-encoder",
					Source: requestEncoderT,
					Data:   e,
					FuncMap: map[string]interface{}{
						"typeConversionData": typeConversionData,
						"isBearer":           isBearer,
					},
				})
			}
			if e.Response.ClientConvert != nil {
				sections = append(sections, &codegen.SectionTemplate{
					Name:    "response-decoder",
					Source:  responseDecoderT,
					Data:    e,
					FuncMap: fm,
				})
			}
		}
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// isBearer returns true if the security scheme uses a Bearer scheme.
func isBearer(schemes []*service.SchemeData) bool {
	for _, s := range schemes {
		if s.Name != "Authorization" {
			continue
		}
		if s.Type == "JWT" || s.Type == "OAuth2" {
			return true
		}
	}
	return false
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
const clientEndpointInitT = `{{ printf "%s calls the %q function in %s.%s interface." .Method.VarName .Method.VarName .PkgName .ClientInterface | comment }}
func (c *{{ .ClientStruct }}) {{ .Method.VarName }}() goa.Endpoint {
	return func(ctx context.Context, v interface{}) (interface{}, error) {
	{{- if .PayloadRef }}
		p, ok := v.({{ .PayloadRef }})
		if !ok {
			return nil, goagrpc.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .PayloadRef }}", v)
    }
		ctx{{ if not .Method.StreamingPayload }}, req :={{ else }} ={{ end }} Encode{{ .Method.VarName }}Request(ctx, p)
	{{- end }}
		{{- if and .Response.Headers .Response.Trailers }}
			var hdr, trlr metadata.MD
		{{- else if .Response.Headers }}
			var hdr metadata.MD
		{{- else if .Response.Trailers }}
			var trlr metadata.MD
		{{- end }}
		{{ if .ClientStream }}stream
		{{- else if .ResultRef }}resp
		{{- else }}_
		{{- end }}, err := c.grpccli.{{ .Method.VarName }}(ctx,
			{{- if not .Method.StreamingPayload }}req, {{ end }}
			{{- if .Response.Headers }}grpc.Header(&hdr), {{ end }}
			{{- if .Response.Trailers }}grpc.Trailer(&trlr), {{ end }}c.opts...)
		if err != nil {
			return nil, err
		}
	{{- if .ClientStream }}
		return &{{ .ClientStream.VarName }}{stream: stream}, nil
	{{- else if .ResultRef }}
		return Decode{{ .Method.VarName }}Response(ctx, resp{{ if .Response.Headers }}, hdr{{ end }}{{ if .Response.Trailers }}, trlr{{ end }})
	{{- else }}
		return nil, nil
	{{- end }}
	}
}
`

// input: EndpointData
const responseDecoderT = `{{ printf "Decode%sResponse decodes responses from the %s %s endpoint." .Method.VarName .ServiceName .Method.Name | comment }}
func Decode{{ .Method.VarName }}Response(ctx context.Context, resp {{ .Response.ServerConvert.TgtRef }}{{ if and .Response.Headers .Response.Trailers }}hdr, trlr metadata.MD{{ else if .Response.Headers }}hdr metadata.MD{{ else if .Response.Trailers }}trlr metadata.MD{{ end }}) ({{ .ResultRef }}, error) {
{{- if .Response.ClientConvert.Init }}
	{{- if or .Response.Headers .Response.Trailers }}
		var (
		{{- range .Response.Headers }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .Response.Trailers }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
			err error
		)
		{
			{{- range .Response.Headers }}
				{{ template "metadata_decoder" (metadataEncodeDecodeData . "hdr") }}
			{{- end }}
			{{- range .Response.Trailers }}
				{{ template "metadata_decoder" (metadataEncodeDecodeData . "trlr") }}
			{{- end }}
		}
		if err != nil {
			return nil, err
		}
	{{- end }}
	res := {{ .Response.ClientConvert.Init.Name }}({{ range .Response.ClientConvert.Init.Args }}{{ .Name }}, {{ end }})
{{- else }}
	res := {{ convertType "resp.Field" . false }}
{{- end }}
	return res, nil
}

{{- define "metadata_decoder" }}
	{{- if or (eq .Metadata.Type.Name "string") (eq .Metadata.Type.Name "any") }}
		{{- if .Metadata.Required }}
			if v := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len(v) == 0 {
				err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Metadata.Name }}, "metadata"))
			} else {
				{{ .Metadata.VarName }} = v[0]
			}
		{{- else }}
			if v := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len(v) > 0 {
				{{ .Metadata.VarName }} = v[0]
			}
		{{- end }}
	{{- else if .Metadata.StringSlice }}
		{{- if .Metadata.Required }}
			if v := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len(v) == 0 {
				err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Metadata.Name }}, "metadata"))
			} else {
				{{ .Metadata.VarName }} = v
			}
		{{- else }}
			{{ .Metadata.VarName }} = {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }})
		{{- end }}
	{{- else if .Metadata.Slice }}
		{{- if .Metadata.Required }}
			if {{ .Metadata.VarName }}Raw := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len({{ .Metadata.VarName }}Raw) == 0 {
				err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Metadata.Name }}, "metadata"))
			} else {
				{{- template "slice_conversion" . }}
			}
		{{- else }}
			if {{ .Metadata.VarName }}Raw := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len({{ .Metadata.VarName }}Raw) > 0 {
				{{- template "slice_conversion" . }}
			}
		{{- end }}
	{{- else }}
		{{- if .Metadata.Required }}
			if v := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len(v) == 0 {
				err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Metadata.Name }}, "metadata"))
			} else {
				{{ .Metadata.VarName }}Raw = v[0]
				{{ template "type_conversion" . }}
			}
		{{- else }}
			if v := {{ .VarName }}.Get({{ printf "%q" .Metadata.Name }}); len(v) > 0 {
				{{ .Metadata.VarName }}Raw = v[0]
				{{ template "type_conversion" . }}
			}
		{{- end }}
	{{- end }}
{{- end }}
` + convertStringToTypeT

// input: EndpointData
const requestEncoderT = `{{ printf "Encode%sRequest encodes requests sent to %s %s endpoint." .Method.VarName .ServiceName .Method.Name | comment }}
func Encode{{ .Method.VarName }}Request(ctx context.Context, p {{ .PayloadRef }}) {{ if not .Method.StreamingPayload }}({{ end }}context.Context{{ if not .Method.StreamingPayload }}, {{ .Request.ClientConvert.TgtRef }}){{ end }} {
{{- if not .Method.StreamingPayload }}
	req := {{ .Request.ClientConvert.Init.Name }}({{ range .Request.ClientConvert.Init.Args }}{{ .Name }}, {{ end }})
{{- end }}
{{- if .Request.Metadata }}
	{{- range .Request.Metadata }}
		{{- if .StringSlice }}
			for _, value := range p.{{ .FieldName }} {
				ctx = metadata.AppendToOutgoingContext(ctx, {{ printf "%q" .Name }}, value)
			}
		{{- else if .Slice }}
			for _, value := range p.{{ .FieldName }} {
				{{ template "type_conversion" (typeConversionData .Type.ElemType.Type "valueStr" "value") }}
				ctx = metadata.AppendToOutgoingContext(ctx, {{ printf "%q" .Name }}, valueStr)
			}
		{{- else }}
			{{- if .Pointer }}
				if p.{{ .FieldName }} != nil {
			{{- end }}
				{{- if (and (eq .Name "Authorization") (isBearer $.MetadataSchemes)) }}
					if !strings.Contains({{ if .Pointer }}*{{ end }}p.{{ .FieldName }}, " ") {
						ctx = metadata.AppendToOutgoingContext(ctx, {{ printf "%q" .Name }},
							"Bearer "+{{ if .Pointer }}*{{ end }}p.{{ .FieldName }})
					} else {
				{{- end }}
					ctx = metadata.AppendToOutgoingContext(ctx, {{ printf "%q" .Name }},
						{{- if eq .Type.Name "bytes" }} string(
						{{- else if not (eq .Type.Name "string") }} fmt.Sprintf("%v",
						{{- end }}
						{{- if .Pointer }}*{{ end }}p.{{ .FieldName }}
						{{- if or (eq .Type.Name "bytes") (not (eq .Type.Name "string")) }})
						{{- end }})
				{{- if (and (eq .Name "Authorization") (isBearer $.MetadataSchemes)) }}
					}
				{{- end }}
			{{- if .Pointer }}
				}
			{{- end }}
		{{- end }}
	{{- end }}
{{- end }}
	return ctx{{ if not .Method.StreamingPayload }}, req{{ end }}
}
` + convertTypeToStringT
