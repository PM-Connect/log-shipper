package api

import (
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/monitoring"
)

func AddRoutes(e *echo.Echo, monitor *monitoring.Monitor, consulAddr string, serviceName string) {
	e.GET("/api/sources", GetSourcesRoute(monitor))
	e.GET("/api/targets", GetTargetsRoute(monitor))
	e.GET("/api/workers", GetWorkersRoute(monitor))

	e.GET("/api/nodes", GetNodesRoute(consulAddr, serviceName))
}
