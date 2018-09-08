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
		Messages []*MessageData
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
		// Method is the data for the underlying method expression.
		Method *service.MethodData
		// PayloadRef is the fully qualified reference to the method payload.
		PayloadRef string
		// ResultRef is the fully qualified reference to the method result.
		ResultRef string
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

		// client side

		// ClientStruct is the name of the gRPC client struct,
		ClientStruct string
		// ClientInterface is the name of the gRPC client interface implemented
		// by the service.
		ClientInterface string
	}

	// MessageData contains the data used to render the code related to a
	// message for a gRPC service. It is used in generating the .proto file.
	// A gRPC request message is computed from the method Payload expression
	// (unless specified explicitly by Message DSL in the gRPC endpoint
	// expression). A gRPC response message is computed from the method Result
	// expression (unless specified explicitly by Message DSL in the gRPC
	// response expression). If the method Payload/Result is empty (not defined)
	// a corresponding message type with empty fields is still generated in the
	// .proto file. NOTE: Codegen could use `google.protobuf.Empty` message type
	// in such cases, but generating an empty message type for the request or
	// response makes it easier (and safer) if the request/response message needs
	// to be changed in the future.
	MessageData struct {
		// Name is the message name.
		Name string
		// Description is the message description.
		Description string
		// VarName is the variable name that holds the definition.
		VarName string
		// Def is the message definition.
		Def string
		// Type is the underlying type.
		Type expr.UserType
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
		// Message is the gRPC request message. It is used in generating
		// .proto file.
		Message *MessageData
		// Metadata is the request metadata.
		Metadata []*MetadataData
		// ServerType is the request data with constructor function to
		// initialize the method payload type from the generated payload type in
		// *.pb.go.
		ServerType *TypeData
		// ClientType is the request data with constructor function to
		// initialize the generated payload type in *.pb.go from the
		// method payload.
		ClientType *TypeData
		// PayloadAttr sets the request message from the specified payload type
		// attribute. This field is set when the design uses Message("name") syntax
		// to set the request message and the payload type is an object.
		PayloadAttr string
	}

	// ResponseData describes a gRPC success or error response.
	ResponseData struct {
		// StatusCode is the return code of the response.
		StatusCode string
		// Description is the response description.
		Description string
		// Message is the gRPC response message. It is used in generating
		// .proto file.
		Message *MessageData
		// Headers is the response header metadata.
		Headers []*MetadataData
		// Trailers is the response trailer metadata.
		Trailers []*MetadataData
		// ServerType is the type data with constructor function to
		// initialize the generated response type in *.pb.go from the
		// method result type.
		ServerType *TypeData
		// ClientType is the type data with constructor function to
		// initialize the method result type from the generated response type in
		// *.pb.go.
		ClientType *TypeData
		// ResultAttr sets the response message from the specified result type
		// attribute. This field is set when the design uses Message("name") syntax
		// to set the response message and the result type is an object.
		ResultAttr string
	}

	// TypeData contains the request/response data and the constructor function
	// to initialize the type.
	// For request type, it contains data to transform gRPC request type to the
	// corresponding payload type (server) and vice versa (client).
	// For response type, it contains data to transform gRPC response type to the
	// corresponding result type (client) and vice versa (server).
	TypeData struct {
		// Name is the type name.
		Name string
		// Ref is the fully qualified reference to the type.
		Ref string
		// Init contains the data required to render the constructor if any.
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
		// CLIArgs is the list of arguments for the command-line client.
		// This is set only for the client side.
		CLIArgs []*InitArgData
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
		pkgName = svc.Name + "pb"
		sd = &ServiceData{
			Service:             svc,
			Name:                svc.Name,
			Description:         svc.Description,
			PkgName:             svc.Name + "pb",
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
		// Make request message to a user type
		if _, ok := e.Request.Type.(expr.UserType); !ok {
			e.Request.Type = &expr.UserTypeExpr{
				AttributeExpr: wrapAttr(e.Request),
				TypeName:      fmt.Sprintf("%sRequest", ProtoBufify(e.Name(), true)),
			}
		} else if e.Request.Type == expr.Empty {
			// empty type should still generate a message. Rename the type to have
			// the endpoint name suffixed with Request.
			e.Request.Type = &expr.UserTypeExpr{
				AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
				TypeName:      fmt.Sprintf("%sRequest", ProtoBufify(e.Name(), true)),
			}
		}

		// Make response message to a user type
		if _, ok := e.Response.Message.Type.(expr.UserType); !ok {
			e.Response.Message.Type = &expr.UserTypeExpr{
				AttributeExpr: wrapAttr(e.Response.Message),
				TypeName:      fmt.Sprintf("%sResponse", ProtoBufify(e.Name(), true)),
			}
		} else if e.Response.Message.Type == expr.Empty {
			// empty type should still generate a message. Rename the type to have
			// the endpoint name suffixed with Response.
			e.Response.Message.Type = &expr.UserTypeExpr{
				AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{}},
				TypeName:      fmt.Sprintf("%sResponse", ProtoBufify(e.Name(), true)),
			}
		}

		// collect all the nested messages and return the top-level message
		collect := func(att *expr.AttributeExpr) *MessageData {
			msgs := collectMessages(att, seen, svc.Scope)
			sd.Messages = append(sd.Messages, msgs...)
			return msgs[0]
		}

		var (
			request    *RequestData
			response   *ResponseData
			errors     []*ErrorData
			payloadRef string
			resultRef  string

			md = svc.Method(e.Name())
		)
		{
			request = &RequestData{
				Message:     collect(e.Request),
				Description: e.Request.Description,
				ServerType:  buildRequestTypeData(e, sd, true),
				ClientType:  buildRequestTypeData(e, sd, false),
				Metadata:    extractMetadata(e.Metadata, e.MethodExpr.Payload, svc.Scope),
			}
			// pass the metadata as arguments to payload constructor in server
			// and also in the client CLI args
			for _, m := range request.Metadata {
				arg := &InitArgData{
					Name:      m.VarName,
					Ref:       m.VarName,
					FieldName: m.FieldName,
					TypeName:  m.TypeName,
					TypeRef:   m.TypeRef,
					Pointer:   m.Pointer,
					Required:  m.Required,
					Example:   m.Example,
				}
				request.ServerType.Init.Args = append(request.ServerType.Init.Args, arg)
				request.ClientType.Init.CLIArgs = append(request.ClientType.Init.CLIArgs, arg)
			}
			if e.MethodExpr.Payload.Type != expr.Empty {
				payloadRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			}

			response = &ResponseData{
				Message:     collect(e.Response.Message),
				StatusCode:  statusCodeToGRPCConst(e.Response.StatusCode),
				Description: e.Response.Description,
				ServerType:  buildResponseTypeData(e, sd, true),
				ClientType:  buildResponseTypeData(e, sd, false),
				Headers:     extractMetadata(e.Response.Headers, e.MethodExpr.Result, svc.Scope),
				Trailers:    extractMetadata(e.Response.Trailers, e.MethodExpr.Result, svc.Scope),
			}
			// pass header metadata as arguments to result constructor in client
			for _, m := range response.Headers {
				response.ClientType.Init.Args = append(response.ClientType.Init.Args, &InitArgData{
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
			// pass trailer metadata as arguments to result constructor in client
			for _, m := range response.Trailers {
				response.ClientType.Init.Args = append(response.ClientType.Init.Args, &InitArgData{
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
			if e.MethodExpr.Result.Type != expr.Empty {
				resultRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
			}
			errors = buildErrorsData(e, sd)
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
		sd.Endpoints = append(sd.Endpoints, &EndpointData{
			ServiceName:     svc.Name,
			PkgName:         sd.PkgName,
			Method:          md,
			PayloadRef:      payloadRef,
			ResultRef:       resultRef,
			Request:         request,
			Response:        response,
			MessageSchemes:  msgSch,
			MetadataSchemes: metSch,
			Errors:          errors,
			ServerStruct:    sd.ServerStruct,
			ServerInterface: sd.ServerInterface,
			ClientStruct:    sd.ClientStruct,
			ClientInterface: sd.ClientInterface,
		})
	}
	return sd
}

// wrapAttr wraps the given attribute into an attribute named "field" if
// the given attribute is a non-object type. For a raw object type it simply
// returns a dupped attribute.
func wrapAttr(att *expr.AttributeExpr) *expr.AttributeExpr {
	var attr *expr.AttributeExpr
	switch actual := att.Type.(type) {
	case *expr.Array:
	case *expr.Map:
	case expr.Primitive:
		attr = &expr.AttributeExpr{
			Type: &expr.Object{
				&expr.NamedAttributeExpr{
					Name: "field",
					Attribute: &expr.AttributeExpr{
						Type: actual,
						Meta: expr.MetaExpr{"rpc:tag": []string{"1"}},
					},
				},
			},
		}
	case *expr.Object:
		attr = expr.DupAtt(att)
	}
	return attr
}

// collectMessages recurses through the attribute to gather all the messages.
func collectMessages(at *expr.AttributeExpr, seen map[string]struct{}, scope *codegen.NameScope) (data []*MessageData) {
	if at == nil {
		return
	}
	collect := func(at *expr.AttributeExpr) []*MessageData { return collectMessages(at, seen, scope) }
	switch dt := at.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.Name()]; ok {
			return nil
		}
		data = append(data, &MessageData{
			Name:        dt.Name(),
			VarName:     ProtoBufMessageName(at, scope),
			Description: dt.Attribute().Description,
			Def:         ProtoBufMessageDef(dt.Attribute(), scope),
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

// buildRequestTypeData builds the type data and the constructor function
// for the server and client packages. It assumes that the gRPC request type
// is not nil.
//	* server side - initializes method payload type from the generated gRPC
//									request type in *.pb.go and the gRPC metadata.
//	* client side - initializes generated gRPC request type in *.pb.go from
//									the method payload.
//
// svr param indicates that the type data is generated for server side.
func buildRequestTypeData(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *TypeData {
	buildInitFn := func(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *InitData {
		if (svr && !needInit(e.MethodExpr.Payload.Type)) ||
			(!svr && e.MethodExpr.Payload.Type == expr.Empty) {
			return nil
		}
		var (
			name     string
			desc     string
			code     string
			retRef   string
			args     []*InitArgData
			srcVar   string
			srcPkg   string
			tgtPkg   string
			srcAtt   *expr.AttributeExpr
			tgtAtt   *expr.AttributeExpr
			cliArgs  []*InitArgData
			isStruct bool

			svc    = sd.Service
			tgtVar = "v"
		)
		{
			if svr {
				name = "New" + svc.Scope.GoTypeName(e.MethodExpr.Payload)
				desc = fmt.Sprintf("%s builds the payload of the %q endpoint of the %q service from the gRPC request type.", name, e.Name(), svc.Name)
				srcAtt = e.Request
				tgtAtt = e.MethodExpr.Payload
				srcVar = "message"
				srcPkg = sd.PkgName
				tgtPkg = svc.PkgName
				retRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Payload, svc.PkgName)
			} else {
				name = "New" + svc.Scope.GoTypeName(e.Request)
				desc = fmt.Sprintf("%s builds the gRPC request type from the payload of the %q endpoint of the %q service.", name, e.Name(), svc.Name)
				srcAtt = e.MethodExpr.Payload
				tgtAtt = e.Request
				srcVar = "p"
				srcPkg = svc.PkgName
				tgtPkg = sd.PkgName
				retRef = ProtoBufFullTypeRef(e.Request, sd.PkgName, svc.Scope)
			}
			isStruct = expr.IsObject(tgtAtt.Type)
			code = protoBufTypeTransformHelper(srcAtt, tgtAtt, srcVar, tgtVar, srcPkg, tgtPkg, !svr, sd)
			args = []*InitArgData{
				&InitArgData{
					Name:     srcVar,
					Ref:      srcVar,
					TypeName: svc.Scope.GoFullTypeName(srcAtt, srcPkg),
					TypeRef:  svc.Scope.GoFullTypeRef(srcAtt, srcPkg),
					Example:  srcAtt.Example(expr.Root.API.Random()),
				},
			}
			if !svr {
				// add the request message as the first argument to the CLI
				cliArgs = []*InitArgData{}
				if obj := expr.AsObject(e.Request.Type); len(*obj) > 0 {
					cliArgs = append(cliArgs, &InitArgData{
						Name:     "message",
						Ref:      "message",
						TypeName: svc.Scope.GoFullTypeName(e.Request, sd.PkgName),
						TypeRef:  svc.Scope.GoFullTypeRef(e.Request, sd.PkgName),
						Example:  e.Request.Example(expr.Root.API.Random()),
					})
				}
			}
		}
		return &InitData{
			Name:           name,
			Description:    desc,
			ReturnVarName:  tgtVar,
			ReturnTypeRef:  retRef,
			ReturnIsStruct: isStruct,
			Code:           code,
			Args:           args,
			CLIArgs:        cliArgs,
		}
	}

	var (
		name string
		ref  string

		svc = sd.Service
	)
	{
		name = ProtoBufMessageName(e.Request, svc.Scope)
		ref = ProtoBufFullTypeRef(e.Request, sd.PkgName, svc.Scope)
	}
	return &TypeData{
		Name: name,
		Ref:  ref,
		Init: buildInitFn(e, sd, svr),
	}
}

// buildResponseTypeData builds the type data and the constructor function
// for the server and client packages. It assumes that the gRPC response type
// is not nil.
//	* server side - initializes generated gRPC response type in *.pb.go from
//									the method result type.
//	* client side - initializes method result type from the generated gRPC
//									response type in *.pb.go and response metadata.
//
// svr param indicates that the type data is generated for server side.
func buildResponseTypeData(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *TypeData {
	buildInitFn := func(e *expr.GRPCEndpointExpr, sd *ServiceData, svr bool) *InitData {
		if (svr && e.MethodExpr.Result.Type == expr.Empty) ||
			(!svr && !needInit(e.MethodExpr.Result.Type)) {
			return nil
		}
		var (
			name   string
			desc   string
			code   string
			retRef string
			args   []*InitArgData
			srcVar string
			srcPkg string
			tgtPkg string
			srcAtt *expr.AttributeExpr
			tgtAtt *expr.AttributeExpr

			svc    = sd.Service
			tgtVar = "v"
		)
		{
			if svr {
				name = "New" + svc.Scope.GoTypeName(e.Response.Message)
				desc = fmt.Sprintf("%s builds the gRPC response type from the result of the %q endpoint of the %q service.", name, e.Name(), svc.Name)
				srcVar = "res"
				srcAtt = e.MethodExpr.Result
				tgtAtt = e.Response.Message
				srcPkg = svc.PkgName
				tgtPkg = sd.PkgName
				retRef = ProtoBufFullTypeRef(e.Response.Message, sd.PkgName, svc.Scope)
			} else {
				name = "New" + svc.Scope.GoTypeName(e.MethodExpr.Result)
				desc = fmt.Sprintf("%s builds the result type of the %q endpoint of the %q service from the gRPC response type.", name, e.Name(), svc.Name)
				srcVar = "resp"
				srcAtt = e.Response.Message
				tgtAtt = e.MethodExpr.Result
				srcPkg = sd.PkgName
				tgtPkg = svc.PkgName
				retRef = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
			}
			code = protoBufTypeTransformHelper(srcAtt, tgtAtt, srcVar, tgtVar, srcPkg, tgtPkg, svr, sd)
			args = []*InitArgData{
				&InitArgData{
					Name:     srcVar,
					Ref:      srcVar,
					TypeName: svc.Scope.GoTypeName(srcAtt),
					TypeRef:  svc.Scope.GoFullTypeRef(srcAtt, srcPkg),
					Example:  srcAtt.Example(expr.Root.API.Random()),
				},
			}
		}
		return &InitData{
			Name:          name,
			Description:   desc,
			ReturnVarName: tgtVar,
			ReturnTypeRef: retRef,
			Code:          code,
			Args:          args,
		}
	}

	var (
		name string
		ref  string

		svc = sd.Service
	)
	if svr {
		name = ProtoBufMessageName(e.Response.Message, svc.Scope)
		ref = ProtoBufFullTypeRef(e.Response.Message, sd.PkgName, svc.Scope)
	} else {
		name = svc.Scope.GoTypeName(e.MethodExpr.Result)
		ref = svc.Scope.GoFullTypeRef(e.MethodExpr.Result, svc.PkgName)
	}
	return &TypeData{
		Name: name,
		Ref:  ref,
		Init: buildInitFn(e, sd, svr),
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

// protoBufTypeTransformHelper is a helper function to transform a protocol
// buffer message type to a Go type and vice versa. If src and tgt are of
// different types (i.e. the Payload/Result is a non-user type and
// Request/Response message is always a user type), the function returns the
// code for initializing the types appropriately by making use of the wrapped
// "field" attribute. Use this function in places where
// codegen.ProtoBufTypeTransform needs to be called.
func protoBufTypeTransformHelper(src, tgt *expr.AttributeExpr, srcVar, tgtVar, srcPkg, tgtPkg string, proto bool, sd *ServiceData) string {
	var (
		code string
		err  error
		h    []*codegen.TransformFunctionData

		svc = sd.Service
	)
	if e := isCompatible(src.Type, tgt.Type, srcVar, tgtVar); e == nil {
		code, h, err = ProtoBufTypeTransform(src.Type, tgt.Type, srcVar, tgtVar, srcPkg, tgtPkg, proto, svc.Scope)
		if err != nil {
			fmt.Println(err.Error()) // TBD validate DSL so errors are not possible
			return ""
		}
		sd.TransformHelpers = codegen.AppendHelpers(sd.TransformHelpers, h)
		return code
	}
	if proto {
		// tgt is a protocol buffer message type. src type is wrapped in an
		// attribute called "field" in tgt.
		pbType := ProtoBufFullMessageName(tgt, tgtPkg, svc.Scope)
		code = fmt.Sprintf("%s := &%s{\nField: %s,\n}", tgtVar, pbType, typeConvert(srcVar, src.Type, tgt.Type, proto))
	} else {
		// tgt is a Go type. src is a protocol buffer message type.
		code = fmt.Sprintf("%s := %s\n", tgtVar, typeConvert(srcVar+".Field", src.Type, tgt.Type, proto))
	}
	return code
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

// needInit returns true if and only if the given type is or makes use of user
// types.
func needInit(dt expr.DataType) bool {
	if dt == expr.Empty {
		return false
	}
	switch actual := dt.(type) {
	case expr.Primitive:
		return false
	case *expr.Array:
		return needInit(actual.ElemType.Type)
	case *expr.Map:
		return needInit(actual.KeyType.Type) ||
			needInit(actual.ElemType.Type)
	case *expr.Object:
		for _, nat := range *actual {
			if needInit(nat.Attribute.Type) {
				return true
			}
		}
		return false
	case expr.UserType:
		return true
	default:
		panic(fmt.Sprintf("unknown data type %T", actual)) // bug
	}
}
