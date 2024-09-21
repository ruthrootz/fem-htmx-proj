package main

import (
  "html/template"
  "io"

  "github.com/labstack/echo/v4"
  "github.com/labstack/echo/v4/middleware"
)

type Templates struct {
  templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
  return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
  return &Templates{
    templates: template.Must(template.ParseGlob("views/*.html")),
  }
}

type Count struct {
  Count int
}

func main() {
  e := echo.New()
  e.Use(middleware.Logger())
  e.Renderer = newTemplate()

  count := Count { Count: 0 }
  e.GET("/", func(c echo.Context) error {
    count.Count++
    // "index" refers to the block that I named "index" in the index.html file
    return c.Render(200, "index", count)
  })

  e.Logger.Fatal(e.Start(":8080"))
}

