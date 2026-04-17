// Package httperr provides an RFC 7807 error handler for grpc-gateway.
// It intercepts gRPC status errors and writes application/problem+json responses
// using the github.com/semmidev/problem library.
package httperr

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/semmidev/problem"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// GatewayErrorHandler converts a gRPC error into an RFC 7807 Problem Details
// JSON response. Registered via runtime.WithErrorHandler in the gateway mux.
//
// Field-level validation errors (from errdetails.BadRequest) are surfaced as
// problem extensions so the client receives a structured list of violations:
//
//	{
//	  "type":    "about:blank",
//	  "title":   "Unprocessable Entity",
//	  "status":  422,
//	  "detail":  "validation failed",
//	  "errors":  [{"field": "title", "description": "is required"}]
//	}
func GatewayErrorHandler(
	ctx context.Context,
	_ *runtime.ServeMux,
	_ runtime.Marshaler,
	w http.ResponseWriter,
	_ *http.Request,
	err error,
) {
	st, ok := status.FromError(err)
	if !ok {
		problem.New(problem.InternalServerError,
			problem.WithDetail("unexpected error"),
		).Write(w)
		return
	}

	tmpl := grpcCodeToProblemTemplate(st.Code())
	opts := []problem.Option{problem.WithDetail(st.Message())}

	// Unpack field violations from BadRequest errdetails
	for _, detail := range st.Details() {
		if br, ok := detail.(proto.Message); ok {
			if bad, ok := br.(*errdetails.BadRequest); ok {
				type fieldViolation struct {
					Field       string `json:"field"`
					Description string `json:"description"`
				}
				violations := make([]fieldViolation, 0, len(bad.GetFieldViolations()))
				for _, v := range bad.GetFieldViolations() {
					violations = append(violations, fieldViolation{
						Field:       v.GetField(),
						Description: v.GetDescription(),
					})
				}
				if len(violations) > 0 {
					opts = append(opts, problem.WithExtension("errors", violations))
				}
			}
		}
	}

	problem.New(tmpl, opts...).Write(w)
}

// grpcCodeToProblemTemplate maps a gRPC status code to an RFC 7807 template.
func grpcCodeToProblemTemplate(code codes.Code) problem.TypeTemplate {
	switch code {
	case codes.NotFound:
		return problem.NotFound
	case codes.AlreadyExists:
		return problem.Conflict
	case codes.PermissionDenied:
		return problem.Forbidden
	case codes.Unauthenticated:
		return problem.Unauthorized
	case codes.InvalidArgument:
		return problem.UnprocessableEntity
	case codes.ResourceExhausted:
		return problem.TooManyRequests
	case codes.Unavailable:
		return problem.ServiceUnavailable
	default:
		return problem.InternalServerError
	}
}
