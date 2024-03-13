package notarizer

import (
	"github.com/labstack/echo/v4"
)

const (
	// API route that will be registered with INX
	APIRoute = "/api/notarizer/v1"

	// ParameterHash contains the string that shall be notarized.
	ParameterHash = "hash"

	// Specific routes
	RouteHealth             = "/health"
	RouteCreateNotarization = "/create/:" + ParameterHash
	RouteVerifyNotarization = "/verify"
)

func setupRoutes(e *echo.Echo) {
	e.GET(RouteHealth, getHealth)
	e.POST(RouteCreateNotarization, createNotarization)
	e.POST(RouteVerifyNotarization, verifyNotarization)
}
