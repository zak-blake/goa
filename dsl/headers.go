package dsl

import (
	"reflect"

	"goa.design/goa/eval"
	"goa.design/goa/expr"
)

// Headers describes HTTP request/response or gRPC response headers.
// When used in a HTTP expression, it groups a set of Header expressions and
// makes it possible to list required headers using the Required function.
// When used in a GRPC response expression, it defines the headers to be sent
// in the response metadata.
//
// To define HTTP headers, Headers must appear in an API or Service HTTP
// expression to define request headers common to all the API or service
// methods. Headers may also appear in a method, response or error HTTP
// expression to define the HTTP endpoint request and response headers.
//
// To define gRPC response header metadata, Headers must appear in a GRPC
// response expression.
//
// Headers accepts one argument: Either a function listing the headers (both
// HTTP and gRPC) or a user type which must be an object and whose attributes
// define the headers (only HTTP).
//
// Example:
//
//     // HTTP headers
//
//     var _ = API("cellar", func() {
//         HTTP(func() {
//             Headers(func() {
//                 Header("version:Api-Version", String, "API version", func() {
//                     Enum("1.0", "2.0")
//                 })
//                 Required("version")
//             })
//         })
//     })
//
//     // gRPC response header metadata
//
//     var CreateResult = ResultType("application/vnd.create", func() {
//         Attributes(func() {
//             Field(1, "name", String, "Name of the created resource")
//             Field(2, "href", String, "Href of the created resource")
//         })
//     })
//
//     Method("create", func() {
//         Payload(CreatePayload)
//         Result(CreateResult)
//         GRPC(func() {
//             Response(func() {
//                 Code(CodeOK)
//                 Headers(func() {
//                     Attribute("name") // "name" sent in the header metadata
//                 })
//             })
//         })
//     })
//
func Headers(args interface{}) {
	h := headers(eval.Current())
	if h == nil {
		eval.IncompatibleDSL()
		return
	}
	if fn, ok := args.(func()); ok {
		eval.Execute(fn, h)
		return
	}
	t, ok := args.(expr.UserType)
	if !ok {
		if _, ok := eval.Current().(*expr.GRPCResponseExpr); ok {
			eval.InvalidArgError("function", args)
		} else {
			eval.InvalidArgError("function or type", args)
		}
		return
	}
	if _, ok := eval.Current().(*expr.GRPCResponseExpr); ok {
		eval.InvalidArgError("function", args)
		return
	}
	o := expr.AsObject(t)
	if o == nil {
		eval.ReportError("type must be an object but got %s", reflect.TypeOf(args).Name())
	}
	h.Merge(expr.NewMappedAttributeExpr(&expr.AttributeExpr{Type: o}))
}

// headers returns the mapped attribute containing the headers for the given
// expression if it's either the root, a service or an endpoint - nil otherwise.
func headers(exp eval.Expression) *expr.MappedAttributeExpr {
	switch e := exp.(type) {
	case *expr.RootExpr:
		if e.API.HTTP.Headers == nil {
			e.API.HTTP.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.API.HTTP.Headers
	case *expr.HTTPServiceExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.HTTPEndpointExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.HTTPResponseExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	case *expr.GRPCResponseExpr:
		if e.Headers == nil {
			e.Headers = expr.NewEmptyMappedAttributeExpr()
		}
		return e.Headers
	default:
		return nil
	}
}
