package util

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(ctx *gin.Context, message string, data interface{}) {
	ctx.JSON(http.StatusOK, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func CreatedResponse(ctx *gin.Context, message string, data interface{}) {
	ctx.JSON(http.StatusCreated, Response{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(ctx *gin.Context, statusCode int, message string) {
	ctx.JSON(statusCode, Response{
		Status:  "error",
		Message: message,
	})
}
