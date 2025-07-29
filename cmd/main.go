package main

import (
  "io"
  "fmt"
  "html/template"
  "net/http"
  "strings"
  "database/sql"
  "os"
  "strconv"

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
  Id int
  Url string
  Title string
}

func getPageTitle(url string) (string, error) {
  resp, err := http.Get(url)
  if err != nil {
    return "", fmt.Errorf("failed to make HTTP request: %w\n", err)
  }
  defer resp.Body.Close()
  if resp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("received non-OK HTTP status: %d\n", resp.StatusCode)
  }
  doc, err := html.Parse(resp.Body)
  if err != nil {
    return "", fmt.Errorf("failed to parse HTML: %w\n", err)
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

func newLink(id int, url string) Link {
  title, err := getPageTitle(url)
  if err != nil {
    fmt.Errorf("error getting page title: %v\n", err)
    return Link {
      Id: id,
      Url: url,
      Title: url,
    }
  }
  return Link {
    Id: id,
    Url: url,
    Title: title,
  }
}

type Data struct {
  Links []Link
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
    if err := rows.Scan(&link.Id, &link.Url); err != nil {
      fmt.Errorf("error scanning row: ", err)
      continue
    }
    links = append(links, newLink(link.Id, link.Url))
  }
  if err := rows.Err(); err != nil {
    fmt.Errorf("error during rows iteration: ", err)
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

  data := Data {}

  err := godotenv.Load()
  if err != nil {
    fmt.Errorf("err loading: %v\n", err)
    os.Exit(1)
  }
  dbUrl := os.Getenv("TURSO_URL")
  if dbUrl == "" {
    fmt.Errorf("TURSO_URL environment variable not set\n")
    os.Exit(1)
  }
  authToken := os.Getenv("TURSO_AUTH_TOKEN")
  if authToken != "" {
    dbUrl += "?authToken=" + authToken
  } else {
    fmt.Errorf("TURSO_AUTH_TOKEN environment variable not set\n")
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
    _, err := db.Exec("INSERT INTO link (url) VALUES (?)", url)
    if err != nil {
      fmt.Errorf("failed to insert new url %s\n", url)
    }
    data.Links = queryLinks(db)
    return c.Render(200, "list-webpages", data)
  })

  e.DELETE("/links/:id", func(c echo.Context) error {
    idStr := c.Param("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
      fmt.Errorf("failed to delete link %s\n", idStr)
      return c.String(400, "invalid id")
    }
    _, err = db.Exec("DELETE FROM link WHERE ID == (?)", id)
    if err != nil {
      fmt.Errorf("failed to delete link %s\n", idStr)
      return c.String(400, "invalid id")
    }
    data.Links = queryLinks(db)
    return c.Render(200, "list-webpages", data)
  })

  e.Logger.Fatal(e.Start(":8080"))
}

