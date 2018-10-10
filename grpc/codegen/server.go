package codegen

import (
	"fmt"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/expr"
)

// ServerFiles returns all the server gRPC transport files.
func ServerFiles(genpkg string, root *expr.RootExpr) []*codegen.File {
	svcLen := len(root.API.GRPC.Services)
	fw := make([]*codegen.File, 2*svcLen)
	for i, svc := range root.API.GRPC.Services {
		fw[i] = server(genpkg, svc)
	}
	for i, svc := range root.API.GRPC.Services {
		fw[i+svcLen] = serverEncodeDecode(genpkg, svc)
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

	sections = append(sections, &codegen.SectionTemplate{
		Name:   "server-struct",
		Source: serverStructT,
		Data:   data,
	})
	for _, e := range data.Endpoints {
		if e.ServerStream != nil {
			sections = append(sections, &codegen.SectionTemplate{
				Name:   "server-stream-struct-type",
				Source: streamStructTypeT,
				Data:   e.ServerStream,
			})
		}
	}
	sections = append(sections, &codegen.SectionTemplate{
		Name:   "server-init",
		Source: serverInitT,
		Data:   data,
	})
	for _, e := range data.Endpoints {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "server-grpc-interface",
			Source: serverGRPCInterfaceT,
			Data:   e,
		})
	}
	for _, e := range data.Endpoints {
		if e.ServerStream != nil {
			if e.ServerStream.SendConvert != nil {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "server-stream-send",
					Source: streamSendT,
					Data:   e.ServerStream,
				})
			}
			if e.Method.StreamKind == expr.ClientStreamKind || e.Method.StreamKind == expr.BidirectionalStreamKind {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "server-stream-recv",
					Source: streamRecvT,
					Data:   e.ServerStream,
				})
			}
			if e.ServerStream.MustClose {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "server-stream-close",
					Source: streamCloseT,
					Data:   e.ServerStream,
				})
			}
			if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName == "" {
				sections = append(sections, &codegen.SectionTemplate{
					Name:   "server-stream-set-view",
					Source: streamSetViewT,
					Data:   e.ServerStream,
				})
			}
		}
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// serverEncodeDecode returns the file defining the gRPC server encoding and
// decoding logic.
func serverEncodeDecode(genpkg string, svc *expr.GRPCServiceExpr) *codegen.File {
	path := filepath.Join(codegen.Gendir, "grpc", codegen.SnakeCase(svc.Name()), "server", "encode_decode.go")
	data := GRPCServices.Get(svc.Name())
	title := fmt.Sprintf("%s GRPC server encoders and decoders", svc.Name())
	sections := []*codegen.SectionTemplate{
		codegen.Header(title, "server", []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "strings"},
			{Path: "strconv"},
			{Path: "google.golang.org/grpc"},
			{Path: "google.golang.org/grpc/metadata"},
			{Path: "goa.design/goa", Name: "goa"},
			{Path: "goa.design/goa/grpc", Name: "goagrpc"},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name())), Name: data.Service.PkgName},
			{Path: filepath.Join(genpkg, codegen.SnakeCase(svc.Name()), "views"), Name: data.Service.ViewsPkg},
			{Path: filepath.Join(genpkg, "grpc", codegen.SnakeCase(svc.Name())), Name: svc.Name() + "pb"},
		}),
	}

	for _, e := range data.Endpoints {
		if e.Response.ServerConvert != nil {
			sections = append(sections, &codegen.SectionTemplate{
				Name:   "response-encoder",
				Source: responseEncoderT,
				Data:   e,
				FuncMap: map[string]interface{}{
					"typeConversionData":       typeConversionData,
					"metadataEncodeDecodeData": metadataEncodeDecodeData,
				},
			})
		}
		if e.PayloadRef != "" {
			sections = append(sections, &codegen.SectionTemplate{
				Name:    "request-decoder",
				Source:  requestDecoderT,
				Data:    e,
				FuncMap: transTmplFuncs(svc),
			})
		}
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

func transTmplFuncs(s *expr.GRPCServiceExpr) map[string]interface{} {
	return map[string]interface{}{
		"goTypeRef": func(dt expr.DataType) string {
			return service.Services.Get(s.Name()).Scope.GoTypeRef(&expr.AttributeExpr{Type: dt})
		},
	}
}

// typeConversionData produces the template data suitable for executing the
// "type_conversion" template.
func typeConversionData(dt expr.DataType, varName string, target string) map[string]interface{} {
	return map[string]interface{}{
		"Type":    dt,
		"VarName": varName,
		"Target":  target,
	}
}

// metadataEncodeDecodeData produces the template data suitable for executing the
// "metadata_decoder" and "metadata_encoder" template.
func metadataEncodeDecodeData(md *MetadataData, vname string) map[string]interface{} {
	return map[string]interface{}{
		"Metadata": md,
		"VarName":  vname,
	}
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

// streamStructTypeT renders the server and client struct types that
// implements the client and server service stream interfaces.
// input: StreamData
const streamStructTypeT = `{{ printf "%s implements the %s.%s interface." .VarName .ServiceInterface | comment }}
type {{ .VarName }} struct {
	stream {{ .Interface }}
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
func (s *{{ .ServerStruct }}) {{ .Method.VarName }}(
	{{- if not .ServerStream }}ctx context.Context, {{ end }}
	{{- if not .Method.StreamingPayload }}message {{ .Request.Message.Ref }},{{ end }}
	{{- if .ServerStream }}stream {{ .ServerStream.Interface }}{{ end }}) {{ if .ServerStream }}error{{ else if .Response.Message }}({{ .Response.Message.Ref }},	error{{ if .Response.Message }}){{ end }}{{ end }} {
{{- if .PayloadRef }}
	p, err := Decode{{ .Method.VarName }}Request(
		{{- if .ServerStream }}stream.Context(), {{ if not .Method.StreamingPayload }}message{{ else }}nil{{ end }},
		{{- else }}ctx, message,
		{{- end }})
	if err != nil {
		return {{ if not .ServerStream }}nil, {{ end }}status.Error(codes.InvalidArgument, err.Error())
	}
	payload := p.({{ .PayloadRef }})
{{- end }}
{{- if .ServerStream }}
	ep := &{{ .ServicePkgName }}.{{ .Method.VarName }}EndpointInput{
		Stream: &{{ .ServerStream.VarName }}{stream: stream},
	{{- if .PayloadRef }}
		Payload: payload,
	{{- end }}
	}
{{- end }}
	{{- $newVar := and .ResultRef (not .ServerStream) }}
	{{ if $newVar }}v
	{{- else }}_
	{{- end }}, err {{ if and .PayloadRef (not $newVar) }}={{ else }}:={{ end }} s.endpoints.{{ .Method.VarName }}({{ if .ServerStream }}stream.Context(){{ else }}ctx{{ end }}, {{ if .ServerStream }}ep{{ else }}payload{{ end }})
	if err != nil {
	{{- if .Errors }}
		en, ok := err.(ErrorNamer)
		if !ok {
			return {{ if not .ServerStream }}nil, {{ end }}err
		}
		switch en.ErrorName() {
		{{- range .Errors }}
		case {{ printf "%q" .Name }}:
			return {{ if not $.ServerStream }}nil, {{ end }}status.Error({{ .Response.StatusCode }}, err.Error())
		{{- end }}
		}
	{{- else }}
		return {{ if not .ServerStream }}nil, {{ end }}err
	{{- end }}
	}
	{{- if .ServerStream }}
		return nil
	{{- else }}
		{{- if .Response.ServerConvert }}
			r, err := Encode{{ .Method.VarName }}Response(ctx, v)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			return r.({{ .Response.ServerConvert.TgtRef }}), nil
		{{- else }}
			return nil, nil
		{{- end }}
	{{- end }}
}
`

// input: EndpointData
const requestDecoderT = `{{ printf "Decode%sRequest decodes requests sent to %s %s endpoint." .Method.VarName .ServiceName .Method.Name | comment }}
func Decode{{ .Method.VarName }}Request(ctx context.Context, v interface{}) (interface{}, error) {
	var (
		payload {{ .PayloadRef }}
		err error
	)
	{
{{- if .Request.Metadata }}
		var (
		{{- range .Request.Metadata }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		)
		{
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
		{{- range .Request.Metadata }}
			{{- if or (eq .Type.Name "string") (eq .Type.Name "any") }}
				{{- if .Required }}
					if vals := md.Get({{ printf "%q" .Name }}); len(vals) == 0 {
						err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Name }}, "metadata"))
					} else {
						{{ .VarName }} = vals[0]
					}
				{{- else }}
					if vals := md.Get({{ printf "%q" .Name }}); len(vals) > 0 {
						{{ .VarName }} = vals[0]
					}
				{{- end }}
			{{- else if .StringSlice }}
				{{- if .Required }}
					if vals := md.Get({{ printf "%q" .Name }}); len(vals) == 0 {
						err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Name }}, "metadata"))
					} else {
						{{ .VarName }} = vals
					}
				{{- else }}
					{{ .VarName }} = md.Get({{ printf "%q" .Name }})
				{{- end }}
			{{- else if .Slice }}
				{{- if .Required }}
					if {{ .VarName }}Raw := md.Get({{ printf "%q" .Name }}); len({{ .VarName }}Raw) == 0 {
						err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Name }}, "metadata"))
					} else {
						{{- template "slice_conversion" . }}
					}
				{{- else }}
					if {{ .VarName }}Raw := md.Get({{ printf "%q" .Name }}); len({{ .VarName }}Raw) > 0 {
						{{- template "slice_conversion" . }}
					}
				{{- end }}
			{{- else }}
				{{- if .Required }}
					if vals := md.Get({{ printf "%q" .Name }}); len(vals) == 0 {
						err = goa.MergeErrors(err, goa.MissingFieldError({{ printf "%q" .Name }}, "metadata"))
					} else {
						{{ .VarName }}Raw = vals[0]
						{{ template "type_conversion" . }}
					}
				{{- else }}
					if vals := md.Get({{ printf "%q" .Name }}); len(vals) > 0 {
						{{ .VarName }}Raw = vals[0]
						{{ template "type_conversion" . }}
					}
				{{- end }}
			{{- end }}
		{{- end }}
			}
		}
{{- end }}
{{- if not .Method.StreamingPayload }}
	message, ok := v.({{ .Request.Message.Ref }})
	if !ok {
		return nil, goagrpc.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .Request.Message.Ref }}", v)
	}
{{- end }}
	payload = {{ .Request.ServerConvert.Init.Name }}({{ range .Request.ServerConvert.Init.Args }}{{ .Name }}, {{ end }})
{{- range .MetadataSchemes }}
	{{- if ne .Type "Basic" }}
		{{- if not .CredRequired }}
			if payload.{{ .CredField }} != nil {
		{{- end }}
			if strings.Contains({{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}, " ") {
				// Remove authorization scheme prefix (e.g. "Bearer")
				cred := strings.SplitN({{ if .CredPointer }}*{{ end }}payload.{{ .CredField }}, " ", 2)[1]
				payload.{{ .CredField }} = {{ if .CredPointer }}&{{ end }}cred
			}
		{{- if not .CredRequired }}
		}
		{{- end }}
	{{- end }}
{{- end }}
	}
	return payload, err
}
` + convertStringToTypeT

// input: EndpointData
const responseEncoderT = `{{ printf "Encode%sResponse encodes responses from the %s %s endpoint." .Method.VarName .ServiceName .Method.Name | comment }}
func Encode{{ .Method.VarName }}Response(ctx context.Context, v interface{}) (interface{}, error) {
{{- if .ViewedResultRef }}
	vres, ok := v.({{ .ViewedResultRef }})
	if !ok {
		return nil, goagrpc.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .ViewedResultRef }}", v)
	}
	res := vres.Projected
{{- else }}
	res, ok := v.({{ .ResultRef }})
	if !ok {
		return nil, goagrpc.ErrInvalidType("{{ .ServiceName }}", "{{ .Method.Name }}", "{{ .ResultRef }}", v)
	}
{{- end }}
	resp := {{ .Response.ServerConvert.Init.Name }}({{ range .Response.ServerConvert.Init.Args }}{{ .Name }}, {{ end }})
{{- if or .Response.Headers .ViewedResultRef }}
	hdr := metadata.New(map[string]string{})
	{{- range .Response.Headers }}
		{{ template "metadata_encoder" (metadataEncodeDecodeData . "hdr") }}
	{{- end }}
	{{- if .ViewedResultRef }}
		hdr.Append("goa-view", vres.View)
	{{- end }}
	grpc.SendHeader(ctx, hdr)
{{- end }}
{{- if .Response.Trailers }}
	trlr := metadata.New(map[string]string{})
	{{- range .Response.Trailers }}
		{{ template "metadata_encoder" (metadataEncodeDecodeData . "trlr") }}
	{{- end }}
	grpc.SendTrailer(ctx, trlr)
{{- end }}
	return resp, nil
}

{{- define "metadata_encoder" }}
	{{- if .Metadata.StringSlice }}
	{{ .VarName }}.Append({{ printf "%q" .Metadata.Name }}, res.{{ .Metadata.FieldName }}...)
	{{- else if .Metadata.Slice }}
		for _, value := range res.{{ .Metadata.FieldName }} {
			{{ template "string_conversion" (typeConversionData .Metadata.Type.ElemType.Type "valueStr" "value") }}
			{{ .VarName }}.Append({{ printf "%q" .Metadata.Name }}, valueStr)
		}
	{{- else }}
		{{- if .Metadata.Pointer }}
			if res.{{ .Metadata.FieldName }} != nil {
		{{- end }}
		{{ .VarName }}.Append({{ printf "%q" .Metadata.Name }},
			{{- if eq .Metadata.Type.Name "bytes" }} string(
			{{- else if not (eq .Metadata.Type.Name "string") }} fmt.Sprintf("%v",
			{{- end }}
			{{- if .Metadata.Pointer }}*{{ end }}p.{{ .Metadata.FieldName }}
			{{- if or (eq .Metadata.Type.Name "bytes") (not (eq .Metadata.Type.Name "string")) }})
			{{- end }})
		{{- if .Metadata.Pointer }}
			}
		{{- end }}
	{{- end }}
{{- end }}
` + convertTypeToStringT

// input: TypeData
const convertStringToTypeT = `{{- define "slice_conversion" }}
	{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}Raw))
	for i, rv := range {{ .VarName }}Raw {
		{{- template "slice_item_conversion" . }}
	}
{{- end }}

{{- define "slice_item_conversion" }}
	{{- if eq .Type.ElemType.Type.Name "string" }}
		{{ .VarName }}[i] = rv
	{{- else if eq .Type.ElemType.Type.Name "bytes" }}
		{{ .VarName }}[i] = []byte(rv)
	{{- else if eq .Type.ElemType.Type.Name "int" }}
		v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of integers"))
		}
		{{ .VarName }}[i] = int(v)
	{{- else if eq .Type.ElemType.Type.Name "int32" }}
		v, err2 := strconv.ParseInt(rv, 10, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of integers"))
		}
		{{ .VarName }}[i] = int32(v)
	{{- else if eq .Type.ElemType.Type.Name "int64" }}
		v, err2 := strconv.ParseInt(rv, 10, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of integers"))
		}
		{{ .VarName }}[i] = v
	{{- else if eq .Type.ElemType.Type.Name "uint" }}
		v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of unsigned integers"))
		}
		{{ .VarName }}[i] = uint(v)
	{{- else if eq .Type.ElemType.Type.Name "uint32" }}
		v, err2 := strconv.ParseUint(rv, 10, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of unsigned integers"))
		}
		{{ .VarName }}[i] = int32(v)
	{{- else if eq .Type.ElemType.Type.Name "uint64" }}
		v, err2 := strconv.ParseUint(rv, 10, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of unsigned integers"))
		}
		{{ .VarName }}[i] = v
	{{- else if eq .Type.ElemType.Type.Name "float32" }}
		v, err2 := strconv.ParseFloat(rv, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of floats"))
		}
		{{ .VarName }}[i] = float32(v)
	{{- else if eq .Type.ElemType.Type.Name "float64" }}
		v, err2 := strconv.ParseFloat(rv, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of floats"))
		}
		{{ .VarName }}[i] = v
	{{- else if eq .Type.ElemType.Type.Name "boolean" }}
		v, err2 := strconv.ParseBool(rv)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "array of booleans"))
		}
		{{ .VarName }}[i] = v
	{{- else if eq .Type.ElemType.Type.Name "any" }}
		{{ .VarName }}[i] = rv
	{{- else }}
		// unsupported slice type {{ .Type.ElemType.Type.Name }} for var {{ .VarName }}
	{{- end }}
{{- end }}

{{- define "type_conversion" }}
	{{- if eq .Type.Name "bytes" }}
		{{ .VarName }} = []byte({{.VarName}}Raw)
	{{- else if eq .Type.Name "int" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := int(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = int(v)
		{{- end }}
	{{- else if eq .Type.Name "int32" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := int32(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = int32(v)
		{{- end }}
	{{- else if eq .Type.Name "int64" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "integer"))
		}
		{{ .VarName }} = {{ if .Pointer}}&{{ end }}v
	{{- else if eq .Type.Name "uint" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := uint(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = uint(v)
		{{- end }}
	{{- else if eq .Type.Name "uint32" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := uint32(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = uint32(v)
		{{- end }}
	{{- else if eq .Type.Name "uint64" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{ .VarName }} = {{ if .Pointer }}&{{ end }}v
	{{- else if eq .Type.Name "float32" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 32)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "float"))
		}
		{{- if .Pointer }}
		pv := float32(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = float32(v)
		{{- end }}
	{{- else if eq .Type.Name "float64" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 64)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "float"))
		}
		{{ .VarName }} = {{ if .Pointer }}&{{ end }}v
	{{- else if eq .Type.Name "boolean" }}
		v, err2 := strconv.ParseBool({{ .VarName }}Raw)
		if err2 != nil {
			err = goa.MergeErrors(err, goa.InvalidFieldTypeError({{ printf "%q" .VarName }}, {{ .VarName}}Raw, "boolean"))
		}
		{{ .VarName }} = {{ if .Pointer }}&{{ end }}v
	{{- else }}
		// unsupported type {{ .Type.Name }} for var {{ .VarName }}
	{{- end }}
{{- end }}
`

// input: TypeData
const convertTypeToStringT = `{{- define "string_conversion" }}
	{{- if eq .Type.Name "boolean" -}}
		{{ .VarName }} := strconv.FormatBool({{ .Target }})
	{{- else if eq .Type.Name "int" -}}
		{{ .VarName }} := strconv.Itoa({{ .Target }})
	{{- else if eq .Type.Name "int32" -}}
		{{ .VarName }} := strconv.FormatInt(int64({{ .Target }}), 10)
	{{- else if eq .Type.Name "int64" -}}
		{{ .VarName }} := strconv.FormatInt({{ .Target }}, 10)
	{{- else if eq .Type.Name "uint" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ .Target }}), 10)
	{{- else if eq .Type.Name "uint32" -}}
		{{ .VarName }} := strconv.FormatUint(uint64({{ .Target }}), 10)
	{{- else if eq .Type.Name "uint64" -}}
		{{ .VarName }} := strconv.FormatUint({{ .Target }}, 10)
	{{- else if eq .Type.Name "float32" -}}
		{{ .VarName }} := strconv.FormatFloat(float64({{ .Target }}), 'f', -1, 32)
	{{- else if eq .Type.Name "float64" -}}
		{{ .VarName }} := strconv.FormatFloat({{ .Target }}, 'f', -1, 64)
	{{- else if eq .Type.Name "string" -}}
		{{ .VarName }} := {{ .Target }}
	{{- else if eq .Type.Name "bytes" -}}
		{{ .VarName }} := string({{ .Target }})
	{{- else if eq .Type.Name "any" -}}
		{{ .VarName }} := fmt.Sprintf("%v", {{ .Target }})
	{{- else }}
		// unsupported type {{ .Type.Name }} for field {{ .FieldName }}
	{{- end }}
{{- end }}
`

// streamSendT renders the function implementing the Send method in
// stream interface.
// input: StreamData
const streamSendT = `{{ comment .SendDesc }}
func (s *{{ .VarName }}) {{ .SendName }}(res {{ .SendConvert.SrcRef }}) error {
	v := {{ .SendConvert.Init.Name }}({{ range .SendConvert.Init.Args }}{{ .Name }}, {{ end }})
	return s.stream.{{ .SendName }}(v)
}
`

// streamRecvT renders the function implementing the Recv method in
// stream interface.
// input: StreamData
const streamRecvT = `{{ comment .RecvDesc }}
func (s *{{ .VarName }}) {{ .RecvName }}() ({{ .RecvConvert.TgtRef }}, error) {
	var res {{ .RecvConvert.TgtRef }}
	v, err := s.stream.{{ .RecvName }}()
	if err != nil {
		return res, err
	}
	res = {{ .RecvConvert.Init.Name }}({{ range .RecvConvert.Init.Args }}{{ .Name }}, {{ end }})
	return res, nil
}
`

// streamCloseT renders the function implementing the Close method in
// stream interface.
// input: StreamData
const streamCloseT = `
func (s *{{ .VarName }}) Close() error {
	{{ comment "nothing to do here" }}
	return nil
}
`

// streamSetViewT renders the function implementing the SetView method in
// server stream interface.
// input: StreamData
const streamSetViewT = `{{ printf "SetView sets the view." | comment }}
func (s *{{ .VarName }}) SetView(view string) {
}
`
