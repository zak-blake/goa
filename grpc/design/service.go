package design

import (
	"fmt"

	"goa.design/goa/design"
	"goa.design/goa/eval"
)

type (
	// ServiceExpr describes a gRPC service.
	ServiceExpr struct {
		eval.DSLFunc
		// ServiceExpr is the service expression that backs this service.
		ServiceExpr *design.ServiceExpr
		// Name of parent service if any
		ParentName string
		// GRPCEndpoints is the list of service endpoints.
		GRPCEndpoints []*EndpointExpr
		// GRPCErrors lists gRPC errors that apply to all endpoints.
		GRPCErrors []*ErrorExpr
		// Metadata is a set of key/value pairs with semantic that is
		// specific to each generator.
		Metadata design.MetadataExpr
	}
)

// Name of service (service)
func (svc *ServiceExpr) Name() string {
	return svc.ServiceExpr.Name
}

// Description of service (service)
func (svc *ServiceExpr) Description() string {
	return svc.ServiceExpr.Description
}

// Endpoint returns the service endpoint with the given name or nil if there
// isn't one.
func (svc *ServiceExpr) Endpoint(name string) *EndpointExpr {
	for _, a := range svc.GRPCEndpoints {
		if a.Name() == name {
			return a
		}
	}
	return nil
}

// EndpointFor builds the endpoint for the given method.
func (svc *ServiceExpr) EndpointFor(name string, m *design.MethodExpr) *EndpointExpr {
	if a := svc.Endpoint(name); a != nil {
		return a
	}
	a := &EndpointExpr{
		MethodExpr: m,
		Service:    svc,
	}
	svc.GRPCEndpoints = append(svc.GRPCEndpoints, a)
	return a
}

// Error returns the error with the given name.
func (svc *ServiceExpr) Error(name string) *design.ErrorExpr {
	for _, erro := range svc.ServiceExpr.Errors {
		if erro.Name == name {
			return erro
		}
	}
	return Root.Design.Error(name)
}

// GRPCError returns the service gRPC error with given name if any.
func (svc *ServiceExpr) GRPCError(name string) *ErrorExpr {
	for _, erro := range svc.GRPCErrors {
		if erro.Name == name {
			return erro
		}
	}
	return nil
}

// EvalName returns the generic definition name used in error messages.
func (svc *ServiceExpr) EvalName() string {
	if svc.Name() == "" {
		return "unnamed service"
	}
	return fmt.Sprintf("service %#v", svc.Name())
}

// Prepare initializes the error responses.
func (svc *ServiceExpr) Prepare() {
	for _, er := range svc.GRPCErrors {
		er.Response.Prepare()
	}
}

// Validate makes sure the service is valid.
func (svc *ServiceExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	// Validate errors
	for _, er := range svc.GRPCErrors {
		verr.Merge(er.Validate())
	}
	for _, er := range Root.GRPCErrors {
		// This may result in the same error being validated multiple
		// times however service is the top level expression being
		// walked and errors cannot be walked until all expressions have
		// run. Another solution could be to append a new dynamically
		// generated root that the eval engine would process after. Keep
		// things simple for now.
		verr.Merge(er.Validate())
	}
	return verr
}
