package notarizer

import (
	"github.com/labstack/echo/v4"
)

const (
	APIRoute = "/api/inx-notarizer/v1"
)

const (
	RouteHealth             = "/health"
	RouteCreateNotarization = "/create/"
	RouteVerifyNotarization = "/verify"
)

func setupRoutes(e *echo.Echo) {
	e.GET(RouteHealth, getHealth)
	e.POST(RouteCreateNotarization+":hash", createNotarization)
	e.POST(RouteVerifyNotarization, verifyNotarization)
}
