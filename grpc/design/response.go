package design

import (
	"fmt"

	"goa.design/goa/design"
	"goa.design/goa/eval"
)

type (
	// GRPCResponseExpr defines a gRPC response including its status code
	// and result type.
	GRPCResponseExpr struct {
		// gRPC status code
		StatusCode Code
		// Response description
		Description string
		// Response Message if any
		Message *design.AttributeExpr
		// Parent expression, one of EndpointExpr, ServiceExpr or
		// RootExpr.
		Parent eval.Expression
		// Metadata is a list of key/value pairs
		Metadata design.MetadataExpr
	}

	// Code is the error code as defined in the gRPC.
	// See https://github.com/grpc/grpc-go/blob/master/codes/codes.go for more
	// information about supported gRPC error codes.
	Code int
)

const (
	StatusOK Code = iota + 1
	StatusCanceled
	StatusUnknown
	StatusInvalidArgument
	StatusDeadlineExceeded
	StatusNotFound
	StatusAlreadyExists
	StatusPermissionDenied
	StatusResourceExhausted
	StatusFailedPrecondition
	StatusAborted
	StatusOutOfRange
	StatusUnimplemented
	StatusInternal
	StatusUnavailable
	StatusDataLoss
	StatusUnauthenticated
)

// EvalName returns the generic definition name used in error messages.
func (r *GRPCResponseExpr) EvalName() string {
	var suffix string
	if r.Parent != nil {
		suffix = fmt.Sprintf(" of %s", r.Parent.EvalName())
	}
	return "gRPC response" + suffix
}

// Prepare makes sure the response is initialized even if not done explicitly
// by design.
func (r *GRPCResponseExpr) Prepare() {
	if r.Message == nil {
		r.Message = &design.AttributeExpr{Type: design.Empty}
	}
}

// Validate checks that the response definition is consistent: its status is set
// and the result type definition if any is valid.
func (r *GRPCResponseExpr) Validate(e *EndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)

	rt, isrt := e.MethodExpr.Result.Type.(*design.ResultTypeExpr)
	var inview string
	if isrt {
		inview = " all views in"
	}
	hasAttribute := func(name string) bool {
		if !design.IsObject(e.MethodExpr.Result.Type) {
			return false
		}
		if !isrt {
			return e.MethodExpr.Result.Find(name) != nil
		}
		if v, ok := e.MethodExpr.Result.Metadata["view"]; ok {
			return rt.ViewHasAttribute(v[0], name)
		}
		for _, v := range rt.Views {
			if !rt.ViewHasAttribute(v.Name, name) {
				return false
			}
		}
		return true
	}
	if r.Message != nil {
		verr.Merge(r.Message.Validate("gRPC response message", r))
		if att, ok := r.Message.Metadata["origin:attribute"]; ok {
			if !hasAttribute(att[0]) {
				verr.Add(r, "message %q has no equivalent attribute in%s result type", att[0], inview)
			}
		} else if mobj := design.AsObject(r.Message.Type); mobj != nil {
			for _, n := range *mobj {
				if !hasAttribute(n.Name) {
					verr.Add(r, "message %q has no equivalent attribute in%s result type", n.Name, inview)
				}
			}
		}
		verr.Merge(validateMessage(r.Message, e.MethodExpr.Result, e, false))
	}
	return verr
}

// Finalize ensures that the response message type is set. If Message DSL is
// used to set the response message then the message type is set by mapping
// the attributes to the method Result expression. If no response message set
// explicitly, the message is set from the method Result expression.
func (r *GRPCResponseExpr) Finalize(a *EndpointExpr, svcAtt *design.AttributeExpr) {
	r.Parent = a

	// Initialize the message attributes (if an object) with the corresponding
	// result attributes.
	svcObj := design.AsObject(svcAtt.Type)
	if r.Message.Type != design.Empty {
		switch actual := r.Message.Type.(type) {
		case design.UserType:
			// overriding method result type with user type
			actual.Finalize()
		case *design.Object:
			// Raw object type. The attributes in the object will be mapped to the
			// attributes in the result type.
			matt := design.NewMappedAttributeExpr(r.Message)
			for _, nat := range *design.AsObject(matt.Type) {
				var required bool
				if svcObj != nil {
					nat.Attribute = svcObj.Attribute(nat.Name)
					required = svcAtt.IsRequired(nat.Name)
				} else {
					nat.Attribute = svcAtt
					required = svcAtt.Type != design.Empty
				}
				if required {
					if r.Message.Validation == nil {
						r.Message.Validation = &design.ValidationExpr{}
					}
					r.Message.Validation.Required = append(r.Message.Validation.Required, nat.Name)
				}
			}
			r.Message = matt.Attribute()
		}
	} else {
		initMessage(r.Message, svcAtt)
	}
}

// Dup creates a copy of the response expression.
func (r *GRPCResponseExpr) Dup() *GRPCResponseExpr {
	return &GRPCResponseExpr{
		StatusCode:  r.StatusCode,
		Description: r.Description,
		Parent:      r.Parent,
		Metadata:    r.Metadata,
		Message:     design.DupAtt(r.Message),
	}
}
