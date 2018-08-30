package codegen

import (
	"fmt"

	"goa.design/goa/grpc/design"
)

// statusCodeToGRPCConst produces the standard name for the given gRPC status
// code. If no standard name exists then the string consisting of the code
// integer value is returned.
func statusCodeToGRPCConst(code design.Code) string {
	if v, ok := statusCodeToConst[code]; ok {
		return fmt.Sprintf("codes.%s", v)
	}
	return fmt.Sprintf("%d", code)
}

var statusCodeToConst = map[design.Code]string{
	design.StatusOK:                 "OK",
	design.StatusCanceled:           "Canceled",
	design.StatusUnknown:            "Unknown",
	design.StatusInvalidArgument:    "InvalidArgument",
	design.StatusDeadlineExceeded:   "DeadlineExceeded",
	design.StatusNotFound:           "NotFound",
	design.StatusAlreadyExists:      "AlreadyExists",
	design.StatusPermissionDenied:   "PermissionDenied",
	design.StatusResourceExhausted:  "ResourceExhausted",
	design.StatusFailedPrecondition: "FailedPrecondition",
	design.StatusAborted:            "Aborted",
	design.StatusOutOfRange:         "OutOfRange",
	design.StatusUnimplemented:      "Unimplemented",
	design.StatusInternal:           "Internal",
	design.StatusUnavailable:        "Unavailable",
	design.StatusDataLoss:           "DataLoss",
	design.StatusUnauthenticated:    "Unauthenticated",
}
