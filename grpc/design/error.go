package design

import (
	"goa.design/goa/design"
	"goa.design/goa/eval"
)

type (
	// ErrorExpr defines a gRPC error response including its name,
	// status, and result type.
	ErrorExpr struct {
		// ErrorExpr is the underlying goa design error expression.
		*design.ErrorExpr
		// Name of error, we need a separate copy of the name to match it
		// up with the appropriate ErrorExpr.
		Name string
		// Response is the corresponding gRPC response.
		Response *GRPCResponseExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (e *ErrorExpr) EvalName() string {
	return "gRPC error " + e.Name
}

// Validate makes sure there is a error expression that matches the gRPC error
// expression.
func (e *ErrorExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	switch p := e.Response.Parent.(type) {
	case *EndpointExpr:
		if p.MethodExpr.Error(e.Name) == nil {
			verr.Add(e, "Error %#v does not match an error defined in the method", e.Name)
		}
	case *ServiceExpr:
		if p.Error(e.Name) == nil {
			verr.Add(e, "Error %#v does not match an error defined in the service", e.Name)
		}
	case *RootExpr:
		if design.Root.Error(e.Name) == nil {
			verr.Add(e, "Error %#v does not match an error defined in the API", e.Name)
		}
	}
	return verr
}

// Finalize looks up the corresponding method error expression.
func (e *ErrorExpr) Finalize(a *EndpointExpr) {
	var ee *design.ErrorExpr
	switch p := e.Response.Parent.(type) {
	case *EndpointExpr:
		ee = p.MethodExpr.Error(e.Name)
	case *ServiceExpr:
		ee = p.Error(e.Name)
	case *RootExpr:
		ee = design.Root.Error(e.Name)
	}
	e.ErrorExpr = ee
	e.Response.Finalize(a, e.AttributeExpr)
}

// Dup creates a copy of the error expression.
func (e *ErrorExpr) Dup() *ErrorExpr {
	return &ErrorExpr{
		ErrorExpr: e.ErrorExpr,
		Name:      e.Name,
		Response:  e.Response.Dup(),
	}
}
