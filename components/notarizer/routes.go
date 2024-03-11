package notarizer

import (
	"github.com/labstack/echo/v4"
)

const (
	APIRoute = "/api/inx-notarizer/v1"
)

func setupRoutes(e *echo.Echo) {
	e.GET("/health", getHealth)
	e.POST("/create/:hash", createNotarization)
	e.GET("/verify", verifyNotarization)
}
