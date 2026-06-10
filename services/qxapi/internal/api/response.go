package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

func JSONError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorBody{Code: code, Message: message},
	})
}

func JSONValidation(c *gin.Context, message string) {
	JSONError(c, http.StatusBadRequest, "VALIDATION_ERROR", message)
}

func JSONUnauthorized(c *gin.Context) {
	JSONError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
}

func JSONConflict(c *gin.Context, message string) {
	JSONError(c, http.StatusConflict, "CONFLICT", message)
}

func JSONInternal(c *gin.Context) {
	JSONError(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}
