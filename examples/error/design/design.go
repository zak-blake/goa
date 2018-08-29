package design

import (
	. "goa.design/goa/design"
	. "goa.design/goa/dsl"
	grpcdsl "goa.design/goa/grpc/dsl"
	httpdesign "goa.design/goa/http/design"
	httpdsl "goa.design/goa/http/dsl"
	"google.golang.org/grpc/codes"
	grpccodes "google.golang.org/grpc/codes"
)

var _ = API("divider", func() {
	Title("Divider Service")
	Description("An example illustrating error handling in goa. See docs/ErrorHandling.md.")
})

var _ = Service("divider", func() {

	// The "div_by_zero" error is defined at the service level and
	// thus may be returned by both "divide" and "integer_divide".
	Error("div_by_zero", ErrorResult, "divizion by zero")

	// The "timeout" error is also defined at the service level.
	Error("timeout", ErrorResult, "operation timed out, retry later.", func() {
		// Timeout indicates an error due to a timeout.
		Timeout()
		// Temporary indicates that the request may be retried.
		Temporary()
	})

	httpdsl.HTTP(func() {
		// Use HTTP status code 400 Bad Request for "div_by_zero"
		// errors.
		httpdsl.Response("div_by_zero", httpdesign.StatusBadRequest)

		// Use HTTP status code 504 Gateway Timeout for "timeout"
		// errors.
		httpdsl.Response("timeout", httpdesign.StatusGatewayTimeout)
	})

	grpcdsl.GRPC(func() {
		// Use gRPC status code "InvalidArgument" for "div_by_zero"
		// errors.
		grpcdsl.Response("div_by_zero", codes.InvalidArgument)

		// Use gRPC status code "DeadlineExceeded" for "timeout"
		// errors.
		grpcdsl.Response("timeout", codes.DeadlineExceeded)
	})

	Method("integer_divide", func() {
		Payload(IntOperands)
		Result(Int)

		// The "has_remainder" error is defined at the method
		// level and is thus specific to "integer_divide".
		Error("has_remainder", ErrorResult, "integer division has remainder")

		httpdsl.HTTP(func() {
			httpdsl.GET("/idiv/{a}/{b}")
			httpdsl.Response(httpdesign.StatusOK)
			httpdsl.Response("has_remainder", httpdesign.StatusExpectationFailed)
		})

		grpcdsl.GRPC(func() {
			grpcdsl.Response(grpccodes.OK)
			grpcdsl.Response("has_remainder", grpccodes.Unknown)
		})
	})

	Method("divide", func() {
		Payload(FloatOperands)
		Result(Float64)

		httpdsl.HTTP(func() {
			httpdsl.GET("/div/{a}/{b}")
			httpdsl.Response(httpdesign.StatusOK)
		})

		grpcdsl.GRPC(func() {
			grpcdsl.Response(grpccodes.OK)
		})
	})
})

var IntOperands = Type("IntOperands", func() {
	Field(1, "a", Int, "Left operand")
	Field(2, "b", Int, "Right operand")
	Required("a", "b")
})

var FloatOperands = Type("FloatOperands", func() {
	Field(1, "a", Float64, "Left operand")
	Field(2, "b", Float64, "Right operand")
	Required("a", "b")
})
