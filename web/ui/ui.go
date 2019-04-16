package ui

//go:generate packr

import (
	"github.com/gobuffalo/packr"
	"github.com/labstack/echo"
	"github.com/pm-connect/log-shipper/web/ui/middleware"
	"html/template"
	"io"
	"net/http"
	"strings"
)

type Template struct {
	templates *template.Template
}

type templateList map[string]string

func AddRoutes(e *echo.Echo) {
	box := packr.NewBox("./dist")
	assets := packr.NewBox("./dist/assets")

	// Register templates.
	renderer := &Template{
		templates: template.Must(loadTemplates(&templateList{
			"index": "index.html",
		}, e, &box)),
	}
	e.Renderer = renderer

	e.Use(middleware.StaticAssetFS("/ui/assets", &assets))

	e.GET("/ui*", func(c echo.Context) error {
		if strings.Contains(c.Request().URL.Path, "assets") {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Render(http.StatusOK, "index", nil)
	})
}

// Render the requested template.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func loadTemplates(list *templateList, e *echo.Echo, box *packr.Box) (*template.Template, error) {
	var t *template.Template

	for name, path := range *list {
		a, err := box.Find(path)

		if err != nil {
			e.Logger.Fatal(err)
		}

		if t == nil {
			t, _ = template.New(name).Parse(string(a))
		} else {
			t.New(name).Parse(string(a))
		}
	}

	return t, nil
}
