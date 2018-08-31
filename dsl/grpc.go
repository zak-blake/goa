package dsl

import (
	"fmt"

	"goa.design/goa/eval"
	"goa.design/goa/expr"
)

const (
	// CodeOK represents the gRPC response code "OK".
	CodeOK = 0
	// CodeCanceled represents the gRPC response code "Canceled".
	CodeCanceled = 1
	// CodeUnknown represents the gRPC response code "Unknown".
	CodeUnknown = 2
	// CodeInvalidArgument represents the gRPC response code "InvalidArgument".
	CodeInvalidArgument = 3
	// CodeDeadlineExceeded represents the gRPC response code "DeadlineExceeded".
	CodeDeadlineExceeded = 4
	// CodeNotFound represents the gRPC response code "NotFound".
	CodeNotFound = 5
	// CodeAlreadyExists represents the gRPC response code "AlreadyExists".
	CodeAlreadyExists = 6
	// CodePermissionDenied represents the gRPC response code "PermissionDenied".
	CodePermissionDenied = 7
	// CodeResourceExhausted represents the gRPC response code "ResourceExhausted".
	CodeResourceExhausted = 8
	// CodeFailedPrecondition represents the gRPC response code "FailedPrecondition".
	CodeFailedPrecondition = 9
	// CodeAborted represents the gRPC response code "Aborted".
	CodeAborted = 10
	// CodeOutOfRange represents the gRPC response code "OutOfRange".
	CodeOutOfRange = 11
	// CodeUnimplemented represents the gRPC response code "Unimplemented".
	CodeUnimplemented = 12
	// CodeInternal represents the gRPC response code "Internal".
	CodeInternal = 13
	// CodeUnavailable represents the gRPC response code "Unavailable".
	CodeUnavailable = 14
	// CodeDataLoss represents the gRPC response code "DataLoss".
	CodeDataLoss = 15
	// CodeUnauthenticated represents the gRPC response code "Unauthenticated".
	CodeUnauthenticated = 16
)

// GRPC defines the gRPC transport specific properties of an API, a service, or
// a single method. In particular, the function defines the mapping between the
// method payload and gRPC request message and metadata. It also defines the
// mapping between the method result and errors and corresponding gRPC response
// messages and metadata.
//
// The functions that appear in GRPC such as Message or Metadata may take
// advantage of the payload or result type attributes respectively. The
// properties of the message attributes inherit the properties of the attributes
// with the same names that appear in the method payload or result types (so
// there's no need to repeat the attribute type, description, validations etc.).
//
// GRPC must appear in API, a Service or a Method expression.
//
// GRPC accepts a single argument which is the defining DSL function.
//
// Example:
//
//    var _ = Service("calculator", func() {
//        Method("add", func() {
//            Description("Add two operands")
//            Payload(func() {
//                 Attribute("left", Int, "Left operand")
//                 Attribute("right", Int, "Right operand")
//                 Attribute("request_id", String, "Unique request ID")
//            })
//            Result(Int)
//
//            GRPC(func() {
//                Metadata("request_id") // Load "request_id" payload attribute
//                                       // from the gRPC request metadata.
//                                       // Other attributes are loaded from the
//                                       // gRPC request message.
//                Response(CodeOK)
//            })
//        })
//    })
//
func GRPC(fns ...func()) {
	if len(fns) > 1 {
		eval.InvalidArgError("zero or one function", fmt.Sprintf("%d functions", len(fns)))
		return
	}
	fn := func() {}
	if len(fns) == 1 {
		fn = fns[0]
	}
	switch actual := eval.Current().(type) {
	case *expr.APIExpr:
		eval.Execute(fn, actual.GRPC)
	case *expr.ServiceExpr:
		res := expr.Root.API.GRPC.ServiceFor(actual)
		res.DSLFunc = fn
	case *expr.MethodExpr:
		res := expr.Root.API.GRPC.ServiceFor(actual.Service)
		act := res.EndpointFor(actual.Name, actual)
		act.DSLFunc = fn
	default:
		eval.IncompatibleDSL()
	}
}

// Message describes a gRPC request or response message.
//
// Message must appear in a Method GRPC expression to define the request message
// or in an Error or Result GRPC expression to define the response message. If
// Message is absent then the message is built using the method payload or
// result type attributes.
//
// Message accepts one argument which describes the shape of the message, it can
// be:
//
//  - The name of an attribute of the method payload or result type. In this
//    case the attribute type describes the shape of the message.
//
//  - A function listing the message attributes. The attributes inherit the
//    properties (description, type, validations etc.) of the payload or
//    result type attributes with identical names.
//
// Assuming the type:
//
//     var CreatePayload = Type("CreatePayload", func() {
//         Attribute("name", String, "Name of account")
//     })
//
// The following:
//
//     Method("create", func() {
//         Payload(CreatePayload)
//         GRPC()
//     })
//
// is equivalent to:
//
//     Method("create", func() {
//         Payload(CreatePayload)
//         GRPC(func() {
//             Message(func() {
//                 Attribute("name")
//             })
//         })
//     })
//
func Message(args ...interface{}) {
	if len(args) == 0 {
		eval.ReportError("not enough arguments, use Message(name), Message(type), Message(func()) or Message(type, func())")
		return
	}

	var (
		ref       *expr.AttributeExpr
		setter    func(*expr.AttributeExpr)
		kind, tgt string
	)

	// Figure out reference type and setter function
	switch e := eval.Current().(type) {
	case *expr.GRPCEndpointExpr:
		ref = e.MethodExpr.Payload
		setter = func(att *expr.AttributeExpr) {
			e.Request = att
		}
		kind = "request"
		tgt = "Payload"
	case *expr.GRPCErrorExpr:
		ref = e.ErrorExpr.AttributeExpr
		setter = func(att *expr.AttributeExpr) {
			if e.Response == nil {
				e.Response = &expr.GRPCResponseExpr{}
			}
			e.Response.Message = att
		}
		kind = "error_" + e.Name
		tgt = "Error " + e.Name
	case *expr.GRPCResponseExpr:
		ref = e.Parent.(*expr.GRPCEndpointExpr).MethodExpr.Result
		setter = func(att *expr.AttributeExpr) {
			e.Message = att
		}
		kind = "response"
		tgt = "Result"
	default:
		eval.IncompatibleDSL()
		return
	}

	// Now initialize target attribute and DSL if any
	var (
		attr *expr.AttributeExpr
		fn   func()
	)
	switch a := args[0].(type) {
	case string:
		if ref.Find(a) == nil {
			eval.ReportError("%q is not found in %s", a, tgt)
			return
		}
		obj := expr.AsObject(ref.Type)
		if obj == nil {
			eval.ReportError("%s must be an object with an attribute with name %#v, got %T", tgt, a, ref.Type)
			return
		}
		attr = obj.Attribute(a)
		if attr == nil {
			eval.ReportError("%s does not have an attribute named %#v", tgt, a)
			return
		}
		attr = expr.DupAtt(attr)
		if attr.Meta == nil {
			attr.Meta = expr.MetaExpr{"origin:attribute": []string{a}}
		} else {
			attr.Meta["origin:attribute"] = []string{a}
		}
	case expr.UserType:
		attr = &expr.AttributeExpr{Type: a}
		if len(args) > 1 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.ReportError("second argument must be a function")
			}
		}
	case func():
		fn = a
		if ref == nil {
			eval.ReportError("Message is set but %s is not defined", tgt)
			return
		}
		attr = ref
	default:
		eval.InvalidArgError("attribute name, user type or DSL", a)
		return
	}

	if fn != nil {
		eval.Execute(fn, attr)
	}
	if attr != nil {
		if attr.Meta == nil {
			attr.Meta = expr.MetaExpr{}
		}
		attr.Meta["grpc:"+kind] = []string{}
		setter(attr)
	}
}