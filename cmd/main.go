package main

import (
  "html/template"
  "io"
  "strings"

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
  return &Templates {
    templates: template.Must(template.ParseGlob("views/*.html")),
  }
}

type Link struct {
  Url string
  PageName string
}

func GetPageTitle(url string) (string, error) {
  resp, err := http.Get(url)
  if err != nil {
    return "", fmt.Errorf("failed to make HTTP request: %w", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
  }
  doc, err := html.Parse(resp.Body)
  if err != nil {
    return "", fmt.Errorf("failed to parse HTML: %w", err)
  }
  var title string
  var f func(*html.Node)
  f = func(n *html.Node) {
    if n.Type == html.ElementNode && n.Data == "title" {
      if n.FirstChild != nil {
        title = n.FirstChild.Data
      }
      return // Found the title, no need to continue traversing
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
      f(c)
    }
  }
  f(doc)
  return strings.TrimSpace(title), nil
}

func newLink(url string) Link {
  title, err := GetPageTitle(url)
  if err != nil {
    fmt.Printf("error getting page title: %v\n", err)
    return Link {
      Url: url,
      Title: url,
    }
  }
  return Link {
    Url: url,
    Title: title,
  }
}

type Data struct {
  Links []Link
}

func newData() Data {
  return Data {
    Links: []Link {
      newLink("https://google.com"),
      newLink("https://hackernews.com"),
    },
  }
}

func main() {
  e := echo.New()
  e.Use(middleware.StaticWithConfig(middleware.StaticConfig {
    Root:   "static",
  }))
  e.Use(middleware.Logger())
  e.Renderer = newTemplate()

  data := newData()

  e.GET("/", func(c echo.Context) error {
    // "index" refers to the block that I named "index" in the index.html file
    return c.Render(200, "index", data)
  })

  e.POST("/links", func(c echo.Context) error {
    url := c.FormValue("url")
    //if !strings.HasPrefix(url, "https://") {
      //url = "https://" + url
    //}
    //if !strings.Contains(url, ".") {
      //url = url + ".com"
    //}
    data.Links = append(data.Links, newLink(url))
    return c.Render(200, "list-webpages", data)
  })

  e.Logger.Fatal(e.Start(":8080"))
}

// TODO:
// - get name of website and use that for display value

