package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"text/template"

	md "github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
)

const (
	html = `<html>
	<head>
		<meta charset="utf-8">
		<link href="/static/gfm.css" media="all" rel="stylesheet" type="text/css"/>
		<link href="//cdnjs.cloudflare.com/ajax/libs/octicons/2.1.2/octicons.css" media="all" rel="stylesheet" type="text/css"/>
	</head>
	<body>
		<article class="markdown-body entry-content" style="padding: 30px;">
		{{.Content}}
		</article>
	</body>
</html>`
)

var (
	t   = template.Must(template.New("html").Parse(html))
	std = log.New(os.Stderr, "", log.LstdFlags)
)

type (
	ResponseWriter struct{ http.ResponseWriter }
)

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.Header().Set("Status", fmt.Sprintf("%d", statusCode))
	rw.ResponseWriter.WriteHeader(statusCode)
}

func logger(w http.ResponseWriter, r *http.Request) {
	std.Printf("%s %s %s", r.Method, r.URL, w.Header().Get("Status"))
}

func render(w http.ResponseWriter, path string) {
	std.Printf("Rendering %s", path)
	content, err := ioutil.ReadFile(path)
	if err == nil {
		err = t.Execute(w, struct{ Content string }{string(md.Markdown(content))})
	}
	errorHandling(w, err)
}

func errorHandling(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
		data := map[string]interface{}{"Content": "<h1>404: Page Not Found</h1>"}

		if err = t.Execute(w, data); err == nil {
			return
		}
		fallthrough
	default:
		data := map[string]interface{}{
			"Content": fmt.Sprintf("<h1>500 Internal Server Error: %s</h1>", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		if err = t.Execute(w, data); err != nil {
			std.Println(err)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	static := http.StripPrefix("/static", http.FileServer(gfmstyle.Assets))

	switch {
	case strings.HasPrefix(r.RequestURI, "/static/"):
		static.ServeHTTP(w, r)
	case strings.HasSuffix(r.RequestURI, "/"):
		// something has to be done for windows support
		render(w, path.Join(r.RequestURI[1:], "README.md"))
	case strings.HasSuffix(r.RequestURI, ".md"):
		render(w, r.RequestURI[1:])
	default:
		if file, err := os.Open(r.RequestURI[1:]); err != nil {
			errorHandling(w, err)
		} else if _, err := io.Copy(w, file); err != nil {
			errorHandling(w, err)
		}
	}
}

func main() {
	var host string
	var port int
	var directory string

	flag.StringVar(&host, "h", "localhost", "Hostname from which the server will serve request")
	flag.IntVar(&port, "p", 5678, "Port on which the server will serve request")
	flag.Parse()

	if directory = flag.Arg(0); directory == "" {
		directory = "."
	}
	if err := os.Chdir(directory); err != nil {
		std.Fatalln(err)
	}

	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", host, port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w = &ResponseWriter{w}
			handler(w, r)
			logger(w, r)
		}),
		ErrorLog: std,
	}
	std.Printf("Serving directory: %s on %s:%d...", directory, host, port)
	std.Fatalln(server.ListenAndServe())
}
