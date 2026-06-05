package api

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error     ErrorBody `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
}

func ErrorHandler(ctx fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "internal error"
	if fiberErr, ok := err.(*fiber.Error); ok {
		status = fiberErr.Code
		message = fiberErr.Message
	}
	return WriteError(ctx, status, codeFromStatus(status), message)
}

func WriteError(ctx fiber.Ctx, status int, code, message string) error {
	ctx.Status(status)
	response := ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	}
	if requestID, ok := ctx.Locals("requestid").(string); ok && requestID != "" {
		response.RequestID = requestID
	}
	return ctx.JSON(response)
}

func codeFromStatus(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "bad_request"
	case fiber.StatusUnauthorized:
		return "unauthorized"
	case fiber.StatusForbidden:
		return "forbidden"
	case fiber.StatusNotFound:
		return "not_found"
	default:
		if status >= 500 {
			return "internal"
		}
		return strings.ToLower(http.StatusText(status))
	}
}
