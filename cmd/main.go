package main

import (
  "io"
  "fmt"
  "html/template"
  "net/http"
  "strings"
  "database/sql"
  "os"

  _ "github.com/tursodatabase/libsql-client-go/libsql"
  "golang.org/x/net/html"
  "github.com/labstack/echo/v4"
  "github.com/labstack/echo/v4/middleware"
  "github.com/joho/godotenv"
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
  ID int
  Url string
  Title string
}

func getPageTitle(url string) (string, error) {
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
      return
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
      f(c)
    }
  }
  f(doc)
  return strings.TrimSpace(title), nil
}

func newLink(url string) Link {
  title, err := getPageTitle(url)
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

func queryLinks(db *sql.DB) []Link  {
  var links []Link
  rows, err := db.Query("SELECT * FROM link")
  if err != nil {
    fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
    os.Exit(1)
  }
  defer rows.Close()
  for rows.Next() {
    var link Link
    if err := rows.Scan(&link.ID, &link.Url); err != nil {
      fmt.Println("error scanning row:", err)
      continue
    }
    links = append(links, newLink(link.Url))
  }
  if err := rows.Err(); err != nil {
    fmt.Println("error during rows iteration:", err)
  }
  return links
}

func main() {
  e := echo.New()
  e.Use(middleware.StaticWithConfig(middleware.StaticConfig {
    Root:   "static",
  }))
  e.Use(middleware.Logger())
  e.Renderer = newTemplate()

  data := newData()

  err := godotenv.Load()
  if err != nil {
    fmt.Errorf("err loading: %v", err)
    os.Exit(1)
  }
  dbUrl := os.Getenv("TURSO_URL")
  if dbUrl == "" {
    fmt.Errorf("TURSO_URL environment variable not set")
    os.Exit(1)
  }
  authToken := os.Getenv("TURSO_AUTH_TOKEN")
  if authToken != "" {
    dbUrl += "?authToken=" + authToken
  } else {
    fmt.Errorf("TURSO_AUTH_TOKEN environment variable not set")
    os.Exit(1)
  }
  db, err := sql.Open("libsql", dbUrl)
  if err != nil {
    fmt.Fprintf(os.Stderr, "failed to open db %s: %s", dbUrl, err)
    os.Exit(1)
  }
  defer db.Close()
  data.Links = queryLinks(db)

  e.GET("/", func(c echo.Context) error {
    // "index" refers to the block that I named "index" in the index.html file
    return c.Render(200, "index", data)
  })

  e.POST("/links", func(c echo.Context) error {
    url := c.FormValue("url")
    data.Links = append(data.Links, newLink(url))
    return c.Render(200, "list-webpages", data)
  })

  e.Logger.Fatal(e.Start(":8080"))
}

// TODO:
// - [x] create Turso DB
// - [x] link project to Turso
// - [ ] save Links to DB
// - [ ] add X to each list item
// - [ ] remove link on X click

