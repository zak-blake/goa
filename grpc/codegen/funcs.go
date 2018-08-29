package codegen

import (
	"fmt"
	"google.golang.org/grpc/codes"
)

// statusCodeToGRPCConst produces the standard name for the given gRPC status
// code. If no standard name exists then the string consisting of the code
// integer value is returned.
func statusCodeToGRPCConst(code codes.Code) string {
	if v, ok := statusCodeToConst[code]; ok {
		return fmt.Sprintf("codes.%s", v)
	}
	return fmt.Sprintf("%d", code)
}

var statusCodeToConst = map[codes.Code]string{
	codes.OK:                 "OK",
	codes.Canceled:           "Canceled",
	codes.Unknown:            "Unknown",
	codes.InvalidArgument:    "InvalidArgument",
	codes.DeadlineExceeded:   "DeadlineExceeded",
	codes.NotFound:           "NotFound",
	codes.AlreadyExists:      "AlreadyExists",
	codes.PermissionDenied:   "PermissionDenied",
	codes.ResourceExhausted:  "ResourceExhausted",
	codes.FailedPrecondition: "FailedPrecondition",
	codes.Aborted:            "Aborted",
	codes.OutOfRange:         "OutOfRange",
	codes.Unimplemented:      "Unimplemented",
	codes.Internal:           "Internal",
	codes.Unavailable:        "Unavailable",
	codes.DataLoss:           "DataLoss",
	codes.Unauthenticated:    "Unauthenticated",
}
