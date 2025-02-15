package utils

import "github.com/gin-gonic/gin"

type JSONResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data, omitempty"`
}

func RespondJSON(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, JSONResponse{
		Status:  code >= 200 && code < 300,
		Message: message,
		Data:    data,
	})
}

func RespondError(c *gin.Context, code int, err error) {
	c.JSON(code, JSONResponse{
		Status:  false,
		Message: err.Error(),
	})
}
