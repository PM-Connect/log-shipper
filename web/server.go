package web

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pm-connect/log-shipper/monitoring"
	"github.com/pm-connect/log-shipper/web/api"
	"github.com/pm-connect/log-shipper/web/ui"
	log "github.com/sirupsen/logrus"
)

func StartServer(port int, monitor *monitoring.Monitor, includeUi bool) {
	e := echo.New()

	e.HideBanner = true

	e.Use(
		middleware.Gzip(),
	)

	// Routes...
	e.Pre(middleware.RemoveTrailingSlash(), middleware.CORS())

	api.AddRoutes(e, monitor)

	if includeUi {
		ui.AddRoutes(e)
	}

	log.Fatal(e.Start(fmt.Sprintf(":%d", port)))
}
