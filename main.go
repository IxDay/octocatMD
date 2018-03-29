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

	"github.com/fsnotify/fsnotify"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
	"github.com/spf13/hugo/livereload"
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
		<script data-no-instant>
			document.write('<script src="/livereload.js?port={{.Port}}&mindelay=10"></' + 'script>')
		</script>
	</body>
</html>`
)

var (
	t   = template.Must(template.New("html").Parse(html))
	std = log.New(os.Stderr, "", log.LstdFlags)
)

type (
	Server struct {
		Host      string
		Port      int
		Directory string
	}
	dict map[string]interface{}
)

func (s *Server) Render(w http.ResponseWriter, path string) {
	std.Printf("Rendering %s", path)
	content, err := ioutil.ReadFile(path)
	if err == nil {
		err = s.Execute(w, dict{
			"File":    filepath.Base(path),
			"Content": string(github_flavored_markdown.Markdown(content)),
		})
	}
	s.errorHandling(w, err)
}

func (s *Server) Execute(w io.Writer, d dict) error {
	d["Port"] = s.Port
	return t.Execute(w, d)
}

func (s *Server) errorHandling(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		return
	case os.IsNotExist(err):
		w.WriteHeader(http.StatusNotFound)
		data := dict{
			"File":    "File Not Found",
			"Content": "<h1>404: Page Not Found</h1>",
		}

		if err = s.Execute(w, data); err == nil {
			return
		}
		fallthrough
	default:
		data := dict{
			"File":    "Internal Server Error",
			"Content": fmt.Sprintf("<h1>500 Internal Server Error: %s</h1>", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		if err = s.Execute(w, data); err != nil {
			std.Println(err)
		}
	}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	static := http.StripPrefix("/static", http.FileServer(gfmstyle.Assets))
	uri := r.URL.Path

	switch {
	case uri == "/livereload":
		livereload.Handler(w, r)
	case uri == "/livereload.js":
		livereload.ServeJS(w, r)
	case strings.HasPrefix(uri, "/static/"):
		static.ServeHTTP(w, r)
	case strings.HasSuffix(uri, "/"):
		// something has to be done for windows support
		s.Render(w, path.Join(uri[1:], "README.md"))
	case strings.HasSuffix(uri, ".md"):
		s.Render(w, uri[1:])
	default:
		if file, err := os.Open(uri[1:]); err != nil {
			s.errorHandling(w, err)
		} else if _, err := io.Copy(w, file); err != nil {
			s.errorHandling(w, err)
		}
	}
	std.Printf("%s %s %s", r.Method, r.URL, w.Header().Get("Status"))
}

func newWatcher() (watcher *fsnotify.Watcher, err error) {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(filepath.Base(info.Name()), ".") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
	}
	return
}

func (s *Server) Addr() string { return fmt.Sprintf("%s:%d", s.Host, s.Port) }

func (s *Server) Run() error {
	if err := os.Chdir(s.Directory); err != nil {
		return err
	}
	watcher, err := newWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	livereload.Initialize()

	go func() { // start watching events
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					livereload.ForceRefresh()
				}
			case err := <-watcher.Errors:
				std.Println("error:", err)
			}
		}
	}()
	server := &http.Server{
		Addr:     s.Addr(),
		Handler:  s,
		ErrorLog: std,
	}

	std.Printf("Serving directory: %s on %s...", s.Directory, s.Addr())
	return server.ListenAndServe()
}

func main() {
	server := &Server{}

	flag.StringVar(&server.Host, "h", "localhost", "Hostname from which the server will serve request")
	flag.IntVar(&server.Port, "p", 5678, "Port on which the server will serve request")
	flag.Parse()

	if server.Directory = flag.Arg(0); server.Directory == "" {
		server.Directory = "."
	}
	std.Fatalln(server.Run())
}
