package codegen

import (
	"fmt"

	"goa.design/goa/codegen"
	"goa.design/goa/codegen/service"
	"goa.design/goa/expr"
)

// GRPCServices holds the data computed from the design needed to generate the
// transport code of the services.
var GRPCServices = make(ServicesData)

type (
	// ServicesData encapsulates the data computed from the expr.
	ServicesData map[string]*ServiceData

	// ServiceData contains the data used to render the code related to a
	// single service.
	ServiceData struct {
		// Service contains the related service data.
		Service *service.Data
		// PkgName is the name of the generated package in *.pb.go.
		PkgName string
		// Name is the service name.
		Name string
		// Description is the service description.
		Description string
		// Endpoints describes the gRPC service endpoints.
		Endpoints []*EndpointData
		// Messages describes the message data for this service.
		Messages []*service.UserTypeData
		// ServerStruct is the name of the gRPC server struct.
		ServerStruct string
		// ClientStruct is the name of the gRPC client struct,
		ClientStruct string
		// ServerInit is the name of the constructor of the server struct.
		ServerInit string
		// ClientInit is the name of the constructor of the client struct.
		ClientInit string
		// ServerInterface is the name of the gRPC server interface implemented
		// by the service.
		ServerInterface string
		// ClientInterface is the name of the gRPC client interface implemented
		// by the service.
		ClientInterface string
		// ClientInterfaceInit is the name of the client constructor function in
		// the generated pb.go package.
		ClientInterfaceInit string
		// TransformHelpers is the list of transform functions required by the
		// constructors.
		TransformHelpers []*codegen.TransformFunctionData
	}

	// EndpointData contains the data used to render the code related to
	// gRPC endpoint.
	EndpointData struct {
		// ServiceName is the name of the service.
		ServiceName string
		// PkgName is the name of the generated package in *.pb.go.
		PkgName string
		// ServicePkgName is the name of the service package name.
		ServicePkgName string
		// Method is the data for the underlying method expression.
		Method *service.MethodData
		// PayloadRef is the fully qualified reference to the method payload.
		PayloadRef string
		// ResultRef is the fully qualified reference to the method result.
		ResultRef string
		// ViewedResultRef is the fully qualified reference to the viewed result.
		ViewedResultRef string
		// Request is the gRPC request data.
		Request *RequestData
		// Response is the gRPC response data.
		Response *ResponseData
		// MetadataSchemes lists all the security requirement schemes that
		// apply to the method and are encoded in the request metadata.
		MetadataSchemes []*service.SchemeData
		// MessageSchemes lists all the security requirement schemes that
		// apply to the method and are encoded in the request message.
		MessageSchemes []*service.SchemeData
		// Errors describes the method gRPC errors.
		Errors []*ErrorData

		// server side

		// ServerStruct is the name of the gRPC server struct.
		ServerStruct string
		// ServerInterface is the name of the gRPC server interface implemented
		// by the service.
		ServerInterface string
		// ServerStream is the server stream data.
		ServerStream *StreamData

		// client side

		// ClientStruct is the name of the gRPC client struct,
		ClientStruct string
		// ClientInterface is the name of the gRPC client interface implemented
		// by the service.
		ClientInterface string
		// ClientStream is the client stream data.
		ClientStream *StreamData
	}

	// MetadataData describes a gRPC metadata field.
	MetadataData struct {
		// Name is the name of the metadata key.
		Name string
		// AttributeName is the name of the corresponding attribute.
		AttributeName string
		// Description is the metadata description.
		Description string
		// FieldName is the name of the struct field that holds the
		// metadata value if any, empty string otherwise.
		FieldName string
		// VarName is the name of the Go variable used to read or
		// convert the metadata value.
		VarName string
		// TypeName is the name of the type.
		TypeName string
		// TypeRef is the reference to the type.
		TypeRef string
		// Required is true if the metadata is required.
		Required bool
		// Pointer is true if and only the metadata variable is a pointer.
		Pointer bool
		// StringSlice is true if the metadata value type is array of strings.
		StringSlice bool
		// Slice is true if the metadata value type is an array.
		Slice bool
		// MapStringSlice is true if the metadata value type is a map of string
		// slice.
		MapStringSlice bool
		// Map is true if the metadata value type is a map.
		Map bool
		// Type describes the datatype of the variable value. Mainly
		// used for conversion.
		Type expr.DataType
		// Validate contains the validation code if any.
		Validate string
		// DefaultValue contains the default value if any.
		DefaultValue interface{}
		// Example is an example value.
		Example interface{}
	}

	// ErrorData contains the error information required to generate the
	// transport decode (client) and encode (server) code.
	ErrorData struct {
		// StatusCode is the response gRPC status code.
		StatusCode string
		// Name is the error name.
		Name string
		// Ref is a reference to the error type.
		Ref string
		// Response is the error response data.
		Response *ResponseData
	}

	// RequestData describes a gRPC request.
	RequestData struct {
		// Description is the request description.
		Description string
		// Message is the gRPC request message.
		Message *service.UserTypeData
		// Metadata is the request metadata.
		Metadata []*MetadataData
		// ServerConvert is the request data with constructor function to
		// initialize the method payload type from the generated payload type in
		// *.pb.go.
		ServerConvert *ConvertData
		// ClientConvert is the request data with constructor function to
		// initialize the generated payload type in *.pb.go from the
		// method payload.
		ClientConvert *ConvertData
		// CLIArgs is the list of arguments for the command-line client.
		// This is set only for the client side.
		CLIArgs []*InitArgData
	}

	// ResponseData describes a gRPC success or error response.
	ResponseData struct {
		// StatusCode is the return code of the response.
		StatusCode string
		// Description is the response description.
		Description string
		// Message is the gRPC response message.
		Message *service.UserTypeData
		// Headers is the response header metadata.
		Headers []*MetadataData
		// Trailers is the response trailer metadata.
		Trailers []*MetadataData
		// ServerConvert is the type data with constructor function to
		// initialize the generated response type in *.pb.go from the
		// method result type or the projected result type.
		ServerConvert *ConvertData
		// ClientConvert is the type data with constructor function to
		// initialize the method result type or the projected result type
		// from the generated response type in *.pb.go.
		ClientConvert *ConvertData
	}

	// ConvertData contains the data to convert source type to a target type.
	// For request type, it contains data to transform gRPC request type to the
	// corresponding payload type (server) and vice versa (client).
	// For response type, it contains data to transform gRPC response type to the
	// corresponding result type (client) and vice versa (server).
	ConvertData struct {
		// SrcName is the fully qualified name of the source type.
		SrcName string
		// SrcRef is the fully qualified reference to the source type.
		SrcRef string
		// TgtName is the fully qualified name of the target type.
		TgtName string
		// TgtRef is the fully qualified reference to the target type.
		TgtRef string
		// Init contains the data required to render the constructor if any
		// to transform the source type to a target type.
		Init *InitData
	}

	// InitData contains the data required to render a constructor.
	InitData struct {
		// Name is the constructor function name.
		Name string
		// Description is the function description.
		Description string
		// Args is the list of constructor arguments.
		Args []*InitArgData
		// ReturnVarName is the name of the variable to be returned.
		ReturnVarName string
		// ReturnTypeRef is the qualified (including the package name)
		// reference to the return type.
		ReturnTypeRef string
		// ReturnIsStruct is true if the return type is a struct.
		ReturnIsStruct bool
		// Code is the transformation code.
		Code string
	}

	// InitArgData represents a single constructor argument.
	InitArgData struct {
		// Name is the argument name.
		Name string
		// Description is the argument description.
		Description string
		// Reference to the argument, e.g. "&body".
		Ref string
		// FieldName is the name of the data structure field that should
		// be initialized with the argument if any.
		FieldName string
		// TypeName is the argument type name.
		TypeName string
		// TypeRef is the argument type reference.
		TypeRef string
		// Pointer is true if a pointer to the arg should be used.
		Pointer bool
		// Required is true if the arg is required to build the payload.
		Required bool
		// DefaultValue is the default value of the arg.
		DefaultValue interface{}
		// Validate contains the validation code for the argument
		// value if any.
		Validate string
		// Example is a example value
		Example interface{}
	}

	// StreamData contains data to render the stream struct type that implements
	// the service stream interface.
	StreamData struct {
		// VarName is the name of the struct type.
		VarName string
		// Type is the stream type (client or server).
		Type string
		// ServiceInterface is the service interface that the struct implements.
		ServiceInterface string
		// Interface is the stream interface in *.pb.go stored in the struct.
		Interface string
		// Endpoint is the streaming endpoint data.
		Endpoint *EndpointData
		// SendName is the name of the send function.
		SendName string
		// SendDesc is the description for the send function.
		SendDesc string
		// SendRef is the fully	qualified reference to the type sent across the
		// stream.
		SendRef string
		// SendConvert is the type sent through the stream. It contains the
		// constructor to convert the service send type to the type expected by
		// the gRPC send type (in *.pb.go)
		SendConvert *ConvertData
		// RecvConvert is the type received through the stream. It contains the
		// constructor to convert the gRPC type (in *.pb.go) to the service receive
		// type.
		RecvConvert *ConvertData
		// RecvName is the name of the receive function.
		RecvName string
		// RecvDesc is the description for the recv function.
		RecvDesc string
		// RecvRef is the fully	qualified reference to the type received from the
		// stream.
		RecvRef string
		// MustClose indicates whether to generate the Close() function
		// for the stream.
		MustClose bool
	}
)

// Get retrieves the transport data for the service with the given name
// computing it if needed. It returns nil if there is no service with the given
// name.
func (d ServicesData) Get(name string) *ServiceData {
	if data, ok := d[name]; ok {
		return data
	}
	service := expr.Root.API.GRPC.Service(name)
	if service == nil {
		return nil
	}
	d[name] = d.analyze(service)
	return d[name]
}

// Endpoint returns the service method transport data for the endpoint with the
// given name, nil if there isn't one.
func (sd *ServiceData) Endpoint(name string) *EndpointData {
	for _, ed := range sd.Endpoints {
		if ed.Method.Name == name {
			return ed
		}
	}
	return nil
}

// HasUnaryEndpoint returns true if the service has at least one unary endpoint.
func (sd *ServiceData) HasUnaryEndpoint() bool {
	for _, ed := range sd.Endpoints {
		if ed.ServerStream == nil {
			return true
		}
	}
	return false
}

// HasStreamingEndpoint returns true if the service has at least one streaming
// endpoint.
func (sd *ServiceData) HasStreamingEndpoint() bool {
	for _, ed := range sd.Endpoints {
		if ed.ServerStream != nil {
			return true
		}
	}
	return false
}

// analyze creates the data necessary to render the code of the given service.
func (d ServicesData) analyze(gs *expr.GRPCServiceExpr) *ServiceData {
	var (
		sd      *ServiceData
		seen    map[string]struct{}
		svcVarN string
		pkgName string

		svc = service.Services.Get(gs.Name())
	)
	{
		svcVarN = codegen.Goify(svc.Name, true)
		pkgName = "pb"
		sd = &ServiceData{
			Service:             svc,
			Name:                svc.Name,
			Description:         svc.Description,
			PkgName:             pkgName,
			ServerStruct:        "Server",
			ClientStruct:        "Client",
			ServerInit:          "New",
			ClientInit:          "NewClient",
			ServerInterface:     svcVarN + "Server",
			ClientInterface:     svcVarN + "Client",
			ClientInterfaceInit: fmt.Sprintf("%s.New%sClient", pkgName, svcVarN),
		}
		seen = make(map[string]struct{})
	}
	for _, e := range gs.GRPCEndpoints {
		en := protoBufify(e.Name(), true)
		makeProtoBufMessage(e.Request, en+"Request", svc.Scope)
		if e.MethodExpr.StreamingPayload.Type != expr.Empty {
			makeProtoBufMessage(e.StreamingRequest, en+"StreamingRequest", svc.Scope)
		}
		makeProtoBufMessage(e.Response.Message, en+"Response", svc.Scope)

		// collect all the nested messages and return the top-level message
		collect := func(att *expr.AttributeExpr) *service.UserTypeData {
			msgs := collectMessages(att, seen, sd)
			sd.Messages = append(sd.Messages, msgs...)
			return msgs[0]
		}

		var (
			payloadRef      string
			resultRef       string
			viewedResultRef string
			errors          []*ErrorData

			md = svc.Method(e.Name())
		)
		{
			if e.MethodExpr.Payload.Type != expr.Empty {
				payloadRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			}
			if e.MethodExpr.Result.Type != expr.Empty {
				resultRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
			}
			if md.ViewedResult != nil {
				viewedResultRef = md.ViewedResult.FullRef
			}
			errors = buildErrorsData(e, sd)
		}

		var (
			request *RequestData
			reqMD   []*MetadataData
		)
		{
			reqMD = extractMetadata(e.Metadata, e.MethodExpr.Payload, svc.Scope)
			request = &RequestData{
				Description:   e.Request.Description,
				Metadata:      reqMD,
				ServerConvert: buildRequestConvertData(e, reqMD, sd, true),
				ClientConvert: buildRequestConvertData(e, reqMD, sd, false),
			}
			if obj := expr.AsObject(e.Request.Type); len(*obj) > 0 {
				// add the request message as the first argument to the CLI
				request.CLIArgs = append(request.CLIArgs, &InitArgData{
					Name:     "message",
					Ref:      "message",
					TypeName: svc.Scope.GoFullTypeName(e.Request, sd.PkgName),
					TypeRef:  svc.Scope.GoFullTypeRef(e.Request, sd.PkgName),
					Example:  e.Request.Example(expr.Root.API.Random()),
				})
			}
			// pass the metadata as arguments to client CLI args
			for _, m := range reqMD {
				request.CLIArgs = append(request.CLIArgs, &InitArgData{
					Name:      m.VarName,
					Ref:       m.VarName,
					FieldName: m.FieldName,
					TypeName:  m.TypeName,
					TypeRef:   m.TypeRef,
					Pointer:   m.Pointer,
					Required:  m.Required,
					Example:   m.Example,
				})
			}
			if e.StreamingRequest.Type != expr.Empty {
				request.Message = collect(e.StreamingRequest)
			} else {
				request.Message = collect(e.Request)
			}
		}

		var (
			response *ResponseData
			hdrs     []*MetadataData
			trlrs    []*MetadataData
		)
		{
			result := e.MethodExpr.Result
			if md.ViewedResult != nil {
				result = expr.AsObject(md.ViewedResult.Type).Attribute("projected")
			}
			hdrs = extractMetadata(e.Response.Headers, result, svc.Scope)
			trlrs = extractMetadata(e.Response.Trailers, result, svc.Scope)
			response = &ResponseData{
				StatusCode:    statusCodeToGRPCConst(e.Response.StatusCode),
				Description:   e.Response.Description,
				Headers:       hdrs,
				Trailers:      trlrs,
				ServerConvert: buildResponseConvertData(e, hdrs, trlrs, sd, true),
				ClientConvert: buildResponseConvertData(e, hdrs, trlrs, sd, false),
			}
			if e.Response.Message.Type != expr.Empty || !e.MethodExpr.IsStreaming() {
				response.Message = collect(e.Response.Message)
			}
		}

		var (
			msgSch []*service.SchemeData
			metSch []*service.SchemeData
		)
		{
			for _, req := range e.Requirements {
				for _, sch := range req.Schemes {
					s := service.Scheme(md.Requirements, sch.SchemeName).Dup()
					s.In = sch.In
					switch s.In {
					case "message":
						msgSch = service.AppendScheme(msgSch, s)
					default:
						metSch = service.AppendScheme(metSch, s)
					}
				}
			}
		}
		ed := &EndpointData{
			ServiceName:     svc.Name,
			PkgName:         sd.PkgName,
			ServicePkgName:  svc.PkgName,
			Method:          md,
			PayloadRef:      payloadRef,
			ResultRef:       resultRef,
			ViewedResultRef: viewedResultRef,
			Request:         request,
			Response:        response,
			MessageSchemes:  msgSch,
			MetadataSchemes: metSch,
			Errors:          errors,
			ServerStruct:    sd.ServerStruct,
			ServerInterface: sd.ServerInterface,
			ClientStruct:    sd.ClientStruct,
			ClientInterface: sd.ClientInterface,
		}
		sd.Endpoints = append(sd.Endpoints, ed)
		if e.MethodExpr.IsStreaming() {
			ed.ServerStream = buildStreamData(e, sd, true)
			ed.ClientStream = buildStreamData(e, sd, false)
		}
	}
	return sd
}

// collectMessages recurses through the attribute to gather all the messages.
func collectMessages(at *expr.AttributeExpr, seen map[string]struct{}, sd *ServiceData) (data []*service.UserTypeData) {
	if at == nil {
		return
	}
	collect := func(at *expr.AttributeExpr) []*service.UserTypeData { return collectMessages(at, seen, sd) }
	switch dt := at.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.Name()]; ok {
			return nil
		}
		data = append(data, &service.UserTypeData{
			Name:        dt.Name(),
			VarName:     protoBufMessageName(at, sd.Service.Scope),
			Description: dt.Attribute().Description,
			Def:         protoBufMessageDef(dt.Attribute(), sd.Service.Scope),
			Ref:         protoBufGoFullTypeRef(at, sd.PkgName, sd.Service.Scope),
			Type:        dt,
		})
		seen[dt.Name()] = struct{}{}
		data = append(data, collect(dt.Attribute())...)
	case *expr.Object:
		for _, nat := range *dt {
			data = append(data, collect(nat.Attribute)...)
		}
	case *expr.Array:
		data = append(data, collect(dt.ElemType)...)
	case *expr.Map:
		data = append(data, collect(dt.KeyType)...)
		data = append(data, collect(dt.ElemType)...)
	}
	return
}

// buildRequestConvertData builds the convert data for the server and client
// requests.
//	* server side - converts generated gRPC request type in *.pb.go and the
//									gRPC metadata to method payload type.
//	* client side - converts method payload type to generated gRPC request
//									type in *.pb.go.
//
// svr param indicates that the convert data is generated for server side.
func buildRequestConvertData(e *expr.GRPCEndpointExpr, md []*MetadataData, sd *ServiceData, svr bool) *ConvertData {
	// Server-side: No need to build convert data if method payload is empty
	// since server doesn't need to convert incoming message/metadata to payload.
	if (svr && !needInit(e.MethodExpr.Payload.Type)) ||
		// Client-side: No need to build convert data if streaming payload since
		// all attributes in method payload is encoded into request metadata.
		(!svr && e.MethodExpr.IsPayloadStreaming()) {
		return nil
	}
	var (
		td *ConvertData

		svc = sd.Service
	)
	{
		if svr {
			fn := func(data *InitData) *InitData {
				data.Description = fmt.Sprintf("%s builds the payload of the %q endpoint of the %q service from the gRPC request type.", data.Name, e.Name(), svc.Name)
				if e.MethodExpr.StreamingPayload.Type != expr.Empty {
					// If payload streaming, remove the unary message type from the list
					// of constructor args
					data.Args = []*InitArgData{}
				}
				for _, m := range md {
					// pass the metadata as arguments to payload constructor in server
					data.Args = append(data.Args, &InitArgData{
						Name:      m.VarName,
						Ref:       m.VarName,
						FieldName: m.FieldName,
						TypeName:  m.TypeName,
						TypeRef:   m.TypeRef,
						Pointer:   m.Pointer,
						Required:  m.Required,
						Example:   m.Example,
					})
				}
				return data
			}
			td = buildConvertData(e.Request, e.MethodExpr.Payload, "message", "v", sd.PkgName, svc.PkgName, false, sd, fn)
		} else {
			fn := func(data *InitData) *InitData {
				data.Description = fmt.Sprintf("%s builds the gRPC request type from the payload of the %q endpoint of the %q service.", data.Name, e.Name(), svc.Name)
				if e.MethodExpr.StreamingPayload.Type != expr.Empty {
					// If payload streaming, remove the unary message type from the list
					// of constructor args
					data.Args = []*InitArgData{}
				}
				return data
			}
			td = buildConvertData(e.MethodExpr.Payload, e.Request, "p", "v", svc.PkgName, sd.PkgName, true, sd, fn)
		}
	}
	return td
}

// buildResponseConvertData builds the convert data for the server and client
// responses.
//	* server side - converts method result type to generated gRPC response type
//									in *.pb.go
//	* client side - converts generated gRPC response type in *.pb.go and
//									response metadata to method result type.
//
// svr param indicates that the convert data is generated for server side.
func buildResponseConvertData(e *expr.GRPCEndpointExpr, hdrs, trlrs []*MetadataData, sd *ServiceData, svr bool) *ConvertData {
	if e.MethodExpr.IsStreaming() || !needInit(e.MethodExpr.Result.Type) {
		return nil
	}
	var (
		td        *ConvertData
		result    *expr.AttributeExpr
		resultPkg string

		svc = sd.Service
		md  = svc.Method(e.Name())
	)
	{
		result = e.MethodExpr.Result
		resultPkg = svc.PkgName
		proj := ""
		if md.ViewedResult != nil {
			result = expr.AsObject(md.ViewedResult.Type).Attribute("projected")
			resultPkg = md.ViewedResult.ViewsPkg
			proj = "projected "
		}
		if svr {
			fn := func(data *InitData) *InitData {
				data.Description = fmt.Sprintf("%s builds the gRPC response type from the %sresult of the %q endpoint of the %q service.", data.Name, proj, e.Name(), svc.Name)
				return data
			}
			td = buildConvertData(result, e.Response.Message, "res", "v", resultPkg, sd.PkgName, true, sd, fn)
		} else {
			fn := func(data *InitData) *InitData {
				data.Description = fmt.Sprintf("%s builds the %sresult type of the %q endpoint of the %q service from the gRPC response type.", data.Name, proj, e.Name(), svc.Name)
				for _, m := range hdrs {
					// pass the headers as arguments to result constructor in client
					data.Args = append(data.Args, &InitArgData{
						Name:      m.VarName,
						Ref:       m.VarName,
						FieldName: m.FieldName,
						TypeName:  m.TypeName,
						TypeRef:   m.TypeRef,
						Pointer:   m.Pointer,
						Required:  m.Required,
						Example:   m.Example,
					})
				}
				for _, m := range trlrs {
					// pass the trailers as arguments to result constructor in client
					data.Args = append(data.Args, &InitArgData{
						Name:      m.VarName,
						Ref:       m.VarName,
						FieldName: m.FieldName,
						TypeName:  m.TypeName,
						TypeRef:   m.TypeRef,
						Pointer:   m.Pointer,
						Required:  m.Required,
						Example:   m.Example,
					})
				}
				return data
			}
			td = buildConvertData(e.Response.Message, result, "resp", "v", sd.PkgName, resultPkg, false, sd, fn)
		}
	}
	return td
}

// buildConvertData builds the ConvertData to convert src to tgt.
//
// src, tgt are the source and target attributes.
// srcVar, tgtVar are the source and target variable names.
// srcPkg, tgtPkg are the source and target package names.
// proto indicates whether the target type is a protocol buffer type.
// sd is the ServiceData
// fn is a function to further configure the constructor InitData (for adding
// metadata as args and CLI args)
func buildConvertData(src, tgt *expr.AttributeExpr, srcVar, tgtVar, srcPkg, tgtPkg string, proto bool, sd *ServiceData, fn func(*InitData) *InitData) *ConvertData {
	var (
		data    *InitData
		srcName string
		srcRef  string
		tgtName string
		tgtRef  string

		svc = sd.Service
	)
	{
		srcName = protoBufGoFullTypeName(src, srcPkg, svc.Scope)
		srcRef = protoBufGoFullTypeRef(src, srcPkg, svc.Scope)
		tgtName = svc.Scope.GoFullTypeName(tgt, tgtPkg)
		tgtRef = svc.Scope.GoFullTypeRef(tgt, tgtPkg)
		if proto {
			srcName = svc.Scope.GoFullTypeName(src, srcPkg)
			srcRef = svc.Scope.GoFullTypeRef(src, srcPkg)
			tgtName = protoBufGoFullTypeName(tgt, tgtPkg, svc.Scope)
			tgtRef = protoBufGoFullTypeRef(tgt, tgtPkg, svc.Scope)
		}
		var (
			name     string
			isStruct bool
			code     string
			helpers  []*codegen.TransformFunctionData
			args     []*InitArgData
			err      error
		)
		{
			n := svc.Scope.GoTypeName(tgt)
			if expr.IsPrimitive(tgt.Type) {
				n = svc.Scope.GoTypeName(src)
			}
			name = "New" + n
			isStruct = expr.IsObject(tgt.Type)
			code, helpers, err = protoBufTypeTransform(src.Type, tgt.Type, srcVar, tgtVar, srcPkg, tgtPkg, proto, svc.Scope)
			if err != nil {
				fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
				return nil
			}
			sd.TransformHelpers = codegen.AppendHelpers(sd.TransformHelpers, helpers)
			args = []*InitArgData{
				&InitArgData{
					Name:     srcVar,
					Ref:      srcVar,
					TypeName: srcName,
					TypeRef:  srcRef,
					Example:  src.Example(expr.Root.API.Random()),
				},
			}
		}
		data = &InitData{
			Name:           name,
			ReturnVarName:  tgtVar,
			ReturnTypeRef:  tgtRef,
			ReturnIsStruct: isStruct,
			Code:           code,
			Args:           args,
		}
		if data != nil && fn != nil {
			data = fn(data)
		}
	}
	return &ConvertData{
		SrcName: srcName,
		SrcRef:  srcRef,
		TgtName: tgtName,
		TgtRef:  tgtRef,
		Init:    data,
	}
}

// buildErrorsData builds the error data for all the error responses in the
// endpoint expression. The response message for each error response are
// inferred from the method's error expression if not specified explicitly.
func buildErrorsData(e *expr.GRPCEndpointExpr, sd *ServiceData) []*ErrorData {
	var (
		errors []*ErrorData

		svc = sd.Service
	)
	errors = make([]*ErrorData, 0, len(e.GRPCErrors))
	for _, v := range e.GRPCErrors {
		var responseData *ResponseData
		{
			responseData = &ResponseData{
				StatusCode:  statusCodeToGRPCConst(v.Response.StatusCode),
				Description: v.Response.Description,
			}
		}
		errors = append(errors, &ErrorData{
			Name:     v.Name,
			Ref:      svc.Scope.GoFullTypeRef(v.ErrorExpr.AttributeExpr, svc.PkgName),
			Response: responseData,
		})
	}
	return errors
}

// buildStreamData builds the StreamData for the server and client streams.
//
// svr param indicates that the stream data is built for the server.
func buildStreamData(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *StreamData {
	var (
		varn      string
		intName   string
		svcInt    string
		sendName  string
		sendDesc  string
		sendRef   string
		sendType  *ConvertData
		recvName  string
		recvDesc  string
		recvRef   string
		recvType  *ConvertData
		mustClose bool
		typ       string

		svc = sd.Service
		ed  = sd.Endpoint(e.Name())
		md  = ed.Method
	)
	{
		result := e.MethodExpr.Result
		resVar := "res"
		resPkg := svc.PkgName
		if md.ViewedResult != nil {
			result = expr.AsObject(md.ViewedResult.Type).Attribute("projected")
			resVar = "vres"
			resPkg = md.ViewedResult.ViewsPkg
		}
		if svr {
			typ = "server"
			varn = md.ServerStream.VarName
			intName = fmt.Sprintf("%s.%s_%sServer", sd.PkgName, svc.StructName, md.VarName)
			svcInt = fmt.Sprintf("%s.%s", svc.PkgName, md.ServerStream.Interface)
			if result.Type != expr.Empty {
				sendName = md.ServerStream.SendName
				sendRef = ed.ResultRef
				sendType = buildConvertData(result, e.Response.Message, resVar, "v", resPkg, sd.PkgName, true, sd, nil)
			}
			if e.MethodExpr.StreamingPayload.Type != expr.Empty {
				recvName = md.ServerStream.RecvName
				recvRef = svc.Scope.GoFullTypeRef(e.MethodExpr.StreamingPayload, svc.PkgName)
				recvType = buildConvertData(e.StreamingRequest, e.MethodExpr.StreamingPayload, "v", "p", sd.PkgName, svc.PkgName, false, sd, nil)
			}
			mustClose = md.ServerStream.MustClose
		} else {
			typ = "client"
			varn = md.ClientStream.VarName
			intName = fmt.Sprintf("%s.%s_%sClient", sd.PkgName, svc.StructName, md.VarName)
			svcInt = fmt.Sprintf("%s.%s", svc.PkgName, md.ClientStream.Interface)
			if e.MethodExpr.StreamingPayload.Type != expr.Empty {
				sendName = md.ClientStream.SendName
				sendRef = svc.Scope.GoFullTypeRef(e.MethodExpr.StreamingPayload, svc.PkgName)
				sendType = buildConvertData(e.MethodExpr.StreamingPayload, e.StreamingRequest, "res", "v", svc.PkgName, sd.PkgName, true, sd, nil)
			}
			if result.Type != expr.Empty {
				recvName = md.ClientStream.RecvName
				recvRef = ed.ResultRef
				recvType = buildConvertData(e.Response.Message, result, "v", resVar, sd.PkgName, resPkg, false, sd, nil)
			}
			mustClose = md.ClientStream.MustClose
		}
		if sendType != nil {
			sendDesc = fmt.Sprintf("%s streams instances of %q to the %q endpoint gRPC stream.", sendName, sendType.TgtName, md.Name)
		}
		if recvType != nil {
			recvDesc = fmt.Sprintf("%s reads instances of %q from the %q endpoint gRPC stream.", recvName, recvType.SrcName, md.Name)
		}
	}
	return &StreamData{
		VarName:          varn,
		Type:             typ,
		Interface:        intName,
		ServiceInterface: svcInt,
		Endpoint:         ed,
		SendName:         sendName,
		SendDesc:         sendDesc,
		SendRef:          sendRef,
		SendConvert:      sendType,
		RecvName:         recvName,
		RecvDesc:         recvDesc,
		RecvRef:          recvRef,
		RecvConvert:      recvType,
		MustClose:        mustClose,
	}
}

// extractMetadata collects the request/response metadata from the given
// metadata attribute and service type (payload/result).
func extractMetadata(a *expr.MappedAttributeExpr, serviceType *expr.AttributeExpr, scope *codegen.NameScope) []*MetadataData {
	var metadata []*MetadataData
	codegen.WalkMappedAttr(a, func(name, elem string, required bool, c *expr.AttributeExpr) error {
		var (
			varn      string
			fieldName string

			arr = expr.AsArray(c.Type)
			mp  = expr.AsMap(c.Type)
		)
		{
			varn = scope.Unique(codegen.Goify(name, false))
			fieldName = codegen.Goify(name, true)
			if !expr.IsObject(serviceType.Type) {
				fieldName = ""
			}
		}
		metadata = append(metadata, &MetadataData{
			Name:          elem,
			AttributeName: name,
			Description:   c.Description,
			FieldName:     fieldName,
			VarName:       varn,
			Required:      required,
			Type:          c.Type,
			TypeName:      scope.GoTypeName(c),
			TypeRef:       scope.GoTypeRef(c),
			Pointer:       a.IsPrimitivePointer(name, true),
			Slice:         arr != nil,
			StringSlice:   arr != nil && arr.ElemType.Type.Kind() == expr.StringKind,
			Map:           mp != nil,
			MapStringSlice: mp != nil &&
				mp.KeyType.Type.Kind() == expr.StringKind &&
				mp.ElemType.Type.Kind() == expr.ArrayKind &&
				expr.AsArray(mp.ElemType.Type).ElemType.Type.Kind() == expr.StringKind,
			DefaultValue: c.DefaultValue,
			Example:      c.Example(expr.Root.API.Random()),
		})
		return nil
	})
	return metadata
}

// needInit returns false if given type is empty.
func needInit(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	if o := expr.AsObject(dt); o != nil && len(*o) == 0 {
		return false
	}
	return true
}

// streamSendT renders the function implementing the Send method in
// stream interface.
// input: StreamData
const streamSendT = `{{ comment .SendDesc }}
func (s *{{ .VarName }}) {{ .SendName }}(res {{ .SendRef }}) error {
{{- if and .Endpoint.Method.ViewedResult (eq .Type "server") }}
	{{- if .Endpoint.Method.ViewedResult.ViewName }}
		vres := {{ .Endpoint.ServicePkgName }}.{{ .Endpoint.Method.ViewedResult.Init.Name }}(res, {{ printf "%q" .Endpoint.Method.ViewedResult.ViewName }})
	{{- else }}
		vres := {{ .Endpoint.ServicePkgName }}.{{ .Endpoint.Method.ViewedResult.Init.Name }}(res, s.view)
	{{- end }}
{{- end }}
	v := {{ .SendConvert.Init.Name }}({{ if and .Endpoint.Method.ViewedResult (eq .Type "server") }}vres.Projected{{ else }}res{{ end }})
	return s.stream.{{ .SendName }}(v)
}
`

// streamRecvT renders the function implementing the Recv method in
// stream interface.
// input: StreamData
const streamRecvT = `{{ comment .RecvDesc }}
func (s *{{ .VarName }}) {{ .RecvName }}() ({{ .RecvRef }}, error) {
	var res {{ .RecvRef }}
	v, err := s.stream.{{ .RecvName }}()
	if err != nil {
		return res, err
	}
{{- if and .Endpoint.Method.ViewedResult (eq .Type "client") }}
	proj := {{ .RecvConvert.Init.Name }}({{ range .RecvConvert.Init.Args }}{{ .Name }}, {{ end }})
	vres := {{ if not .Endpoint.Method.ViewedResult.IsCollection }}&{{ end }}{{ .Endpoint.Method.ViewedResult.FullName }}{Projected: proj, View: s.view}
	return {{ .Endpoint.ServicePkgName }}.{{ .Endpoint.Method.ViewedResult.ResultInit.Name }}(vres), nil
{{- else }}
	return {{ .RecvConvert.Init.Name }}({{ range .RecvConvert.Init.Args }}{{ .Name }}, {{ end }}), nil
{{- end }}
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
	s.view = view
}
`
