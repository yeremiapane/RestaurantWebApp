package middlewares

import (
    "github.com/gin-gonic/gin"
    "github.com/yeremiapane/restaurant-app/utils"
)

func ReceiptLoggerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Sebelum request
        utils.InfoLogger.Printf("Generating receipt for payment ID: %s", c.Param("payment_id"))

        c.Next()

        // Setelah request
        if c.Writer.Status() == 200 {
            utils.InfoLogger.Printf("Receipt generated successfully for payment ID: %s", c.Param("payment_id"))
        } else {
            utils.ErrorLogger.Printf("Failed to generate receipt for payment ID: %s", c.Param("payment_id"))
        }
    }
}