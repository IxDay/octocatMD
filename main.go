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
	"path/filepath"
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
		<style>
			.boxed-group {
				max-width: 980px;
				margin: auto;
			}
			.boxed-group > h3 {
				padding: 9px 10px 10px;
				margin: 0;
				font-size: 14px;
				line-height: 17px;
				background-color: #f6f8fa;
				border: 1px solid #ddd;
				border-bottom: 0;
				border-radius: 3px 3px 0 0;
			}
			.entry-content > :first-child {
				margin-top: 0 !important;
			}
			.entry-content {
				padding: 30px;
				border: 1px solid #ddd;
				border-radius: 0 0 3px 3px;
			}
			.octicon {
				vertical-align: text-bottom;
			}
		</style>
	</head>
	<body class="markdown-body">
		<div class="boxed-group">
		<h3>
			<svg class="octicon octicon-book" viewBox="0 0 16 16" version="1.1" width="16" height="16" aria-hidden="true">
				<path fill-rule="evenodd" d="M3 5h4v1H3V5zm0 3h4V7H3v1zm0 2h4V9H3v1zm11-5h-4v1h4V5zm0 2h-4v1h4V7zm0 2h-4v1h4V9zm2-6v9c0 .55-.45 1-1 1H9.5l-1 1-1-1H2c-.55 0-1-.45-1-1V3c0-.55.45-1 1-1h5.5l1 1 1-1H15c.55 0 1 .45 1 1zm-8 .5L7.5 3H2v9h6V3.5zm7-.5H9.5l-.5.5V12h6V3z">
				</path>
			</svg>
			{{.File}}
			</h3>
			<article class="markdown-body entry-content">
      {{.Content}}
      </article>
    </div>
	</body>
</html>`
)

var (
	t   = template.Must(template.New("html").Parse(html))
	std = log.New(os.Stderr, "", log.LstdFlags)
)

type (
	ResponseWriter struct{ http.ResponseWriter }
	dict           map[string]interface{}
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
		err = t.Execute(w, dict{
			"File":    filepath.Base(path),
			"Content": string(md.Markdown(content)),
		})
	}
	errorHandling(w, err)
}

func errorHandling(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
		data := dict{
			"File":    "File Not Found",
			"Content": "<h1>404: Page Not Found</h1>",
		}

		if err = t.Execute(w, data); err == nil {
			return
		}
		fallthrough
	default:
		data := dict{
			"File":    "Internal Server Error",
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
