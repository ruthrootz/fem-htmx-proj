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

type Link struct {
  Url string
}

func newLink(url string) Link {
  return Link {
    Url: url,
  }
}

type Data struct {
  Links []Link
}

func newData() Data {
  return Data {
    Links: []Link {
      newLink("www.google.com"),
      newLink("www.hackernews.com"),
    },
  }
}

func main() {
  e := echo.New()
  e.Use(middleware.Logger())
  e.Renderer = newTemplate()

  data := newData()

  e.GET("/", func(c echo.Context) error {
    // "index" refers to the block that I named "index" in the index.html file
    return c.Render(200, "index", data)
  })

  e.POST("/links", func(c echo.Context) error {
    l := newLink(c.FormValue("url"))
    data.Links = append(data.Links, l)
    return c.Render(200, "list-webpages", data)
  })

  e.Logger.Fatal(e.Start(":8080"))
}

// TODOS
// - clear input after url is saved
// - add "https://" if it's not part of the saved string
// - get name of website and use that for display value

