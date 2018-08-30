package design

import (
	"fmt"

	"goa.design/goa/design"
	"goa.design/goa/eval"
)

type (
	// EndpointExpr describes a service endpoint. It embeds a MethodExpr
	// and adds gRPC specific properties.
	EndpointExpr struct {
		eval.DSLFunc
		// MethodExpr is the underlying method expression.
		MethodExpr *design.MethodExpr
		// Service is the parent service.
		Service *ServiceExpr
		// Request is the message passed to the gRPC method.
		Request *design.AttributeExpr
		// Responses is the success gRPC response from the method.
		Response *GRPCResponseExpr
		// GRPCErrors is the list of all the possible error gRPC responses.
		GRPCErrors []*ErrorExpr
		// Metadata is a set of key/value pairs with semantic that is
		// specific to each generator, see dsl.Metadata.
		Metadata design.MetadataExpr
	}
)

// Name of gRPC endpoint
func (e *EndpointExpr) Name() string {
	return e.MethodExpr.Name
}

// Description of gRPC endpoint
func (e *EndpointExpr) Description() string {
	return e.MethodExpr.Description
}

// EvalName returns the generic expression name used in error messages.
func (e *EndpointExpr) EvalName() string {
	var prefix, suffix string
	if e.Name() != "" {
		suffix = fmt.Sprintf("gRPC endpoint %#v", e.Name())
	} else {
		suffix = "unnamed gRPC endpoint"
	}
	if e.Service != nil {
		prefix = e.Service.EvalName() + " "
	}
	return prefix + suffix
}

// Prepare initializes the Request and Response if nil.
func (e *EndpointExpr) Prepare() {
	if e.Request == nil {
		e.Request = &design.AttributeExpr{Type: design.Empty}
	}

	// Make sure there's a default response if none define explicitly
	if e.Response == nil {
		e.Response = &GRPCResponseExpr{StatusCode: StatusOK}
	}

	// Inherit gRPC errors from service and root
	for _, r := range e.Service.GRPCErrors {
		e.GRPCErrors = append(e.GRPCErrors, r.Dup())
	}
	for _, r := range Root.GRPCErrors {
		e.GRPCErrors = append(e.GRPCErrors, r.Dup())
	}

	// Prepare response
	e.Response.Prepare()
	for _, er := range e.GRPCErrors {
		er.Response.Prepare()
	}
}

// Validate validates the endpoint expression by checking if the request
// and responses contains the "rpc:tag" in the metadata. It also makes sure
// that there is only one response per status code.
func (e *EndpointExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	if e.Name() == "" {
		verr.Add(e, "Endpoint name cannot be empty")
	}

	// Validate request
	verr.Merge(e.Request.Validate("gRPC request message", e))
	verr.Merge(validateMessage(e.Request, e.MethodExpr.Payload, e, true))

	// Validate response
	verr.Merge(e.Response.Validate(e))

	// Validate errors
	for _, er := range e.GRPCErrors {
		verr.Merge(er.Validate())
	}
	return verr
}

// Finalize ensures the request and response attributes are initialized.
func (e *EndpointExpr) Finalize() {
	// Finalize request
	initMessage(e.Request, e.MethodExpr.Payload)
	// Finalize response
	e.Response.Finalize(e, e.MethodExpr.Result)
	// Finalize errors
	for _, gerr := range e.GRPCErrors {
		gerr.Finalize(e)
	}
}

// validateMessage validates the gRPC message.
//
// msgAtt is the Request/Response message.
// serviceAtt is the Payload/Result attribute.
// e is the endpoint expression.
// req if true indicates the Request message is validated.
func validateMessage(msgAtt, serviceAtt *design.AttributeExpr, e *EndpointExpr, req bool) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)

	validateRPCTag := func(att *design.AttributeExpr) {
		foundRPC := make(map[string]string)
		for _, nat := range *design.AsObject(att.Type) {
			if tag, ok := nat.Attribute.Metadata["rpc:tag"]; !ok {
				verr.Add(e, "attribute %q does not have \"rpc:tag\" defined in the metadata in type %q", nat.Name, att.Type.Name())
			} else if a, ok := foundRPC[tag[0]]; ok {
				verr.Add(e, "field number %d in attribute %q already exists for attribute %q", tag[0], nat.Name, a)
			} else {
				foundRPC[tag[0]] = nat.Name
			}
		}
	}

	msgKind := "Response"
	serviceKind := "Result"
	if req {
		msgKind = "Request"
		serviceKind = "Payload"
	}

	switch msgType := msgAtt.Type.(type) {
	case design.UserType:
		if msgType == design.Empty {
			if obj := design.AsObject(serviceAtt.Type); obj != nil && len(*obj) > 0 {
				validateRPCTag(serviceAtt)
			}
		} else {
			matt := design.NewMappedAttributeExpr(msgAtt)
			validateRPCTag(matt.AttributeExpr)
		}
	case *design.Object:
		srvcObj := design.AsObject(serviceAtt.Type)
		switch {
		case srvcObj == nil:
			verr.Add(e, "%s is an object type but %s is not an object type or user type", msgKind, serviceKind)
		case len(*srvcObj) == 0:
			verr.Add(e, "%s is defined but %s is not defined in Method", msgKind, serviceKind)
		default:
			matt := design.NewMappedAttributeExpr(msgAtt)
			validateRPCTag(matt.AttributeExpr)
			var found bool
			for _, nat := range *design.AsObject(matt.Type) {
				found = false
				for _, snat := range *srvcObj {
					if nat.Name == snat.Name {
						found = true
						break
					}
				}
				if !found {
					verr.Add(e, "%s %q is not found in %s", msgKind, nat.Name, serviceKind)
				}
			}
		}
	default:
		verr.Add(e, "%s is not an object or a user type", msgKind)
	}
	return verr
}

// initMessage initializes the message attribute from the src attribute.
// src may be method Payload or Result expression.
func initMessage(msg *design.AttributeExpr, src *design.AttributeExpr) {
	if msg.Type == design.Empty {
		initAttrFromDesign(msg, src)
		return
	}
	matt := design.NewMappedAttributeExpr(msg)
	srcobj := design.AsObject(src.Type)
	for _, nat := range *design.AsObject(matt.Type) {
		initAttrFromDesign(nat.Attribute, srcobj.Attribute(nat.Name))
	}
	msg = matt.Attribute()
}

// initAttrFromDesign overrides the type of att with the one of patt and
// initializes other non-initialized fields of att with the one of patt except
// Metadata.
func initAttrFromDesign(att, patt *design.AttributeExpr) {
	if patt == nil || patt.Type == design.Empty {
		return
	}
	att.Type = patt.Type
	if att.Description == "" {
		att.Description = patt.Description
	}
	if att.Docs == nil {
		att.Docs = patt.Docs
	}
	if att.Validation == nil {
		att.Validation = patt.Validation
	}
	if att.DefaultValue == nil {
		att.DefaultValue = patt.DefaultValue
	}
	if att.UserExamples == nil {
		att.UserExamples = patt.UserExamples
	}
	if att.DefaultValue == nil {
		att.DefaultValue = patt.DefaultValue
	}
}
