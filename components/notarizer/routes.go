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

func setupRoutes(routeGroup *echo.Group) {
	routeGroup.GET(RouteHealth, getHealth)
	routeGroup.POST(RouteCreateNotarization, createNotarization)
	routeGroup.POST(RouteVerifyNotarization, verifyNotarization)
}
