package middleware

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"net/http"
	"strings"

	"github.com/labstack/echo"
)

func StaticAssetFS(urlPrefix string, box *packr.Box) echo.MiddlewareFunc {
	return serve(urlPrefix, box)
}

func serve(urlPrefix string, box *packr.Box) echo.MiddlewareFunc {
	fileserver := http.FileServer(box)

	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			w, r := c.Response(), c.Request()

			if box.Has(strings.TrimPrefix(r.URL.Path, fmt.Sprintf("%s/", urlPrefix))) {
				fileserver.ServeHTTP(w, r)
				return nil
			}

			if err := next(c); err != nil {
				c.Error(err)
			}

			return nil
		}
	}
}
