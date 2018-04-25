package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/shurcooL/github_flavored_markdown"
	"github.com/shurcooL/github_flavored_markdown/gfmstyle"
	"github.com/spf13/hugo/livereload"
)

type (
	Server struct {
		Host      string
		Port      int
		Directory string
		Renderer
		Logger
	}
	dict = map[string]interface{}
)

func (s *Server) RenderFile(w http.ResponseWriter, path string) {
	s.Log(INFO, "Rendering %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.Render404(w, path)
		} else {
			s.Render500(w, err)
		}
		return
	}
	content = github_flavored_markdown.Markdown(content)
	if err := s.Render(w, filepath.Base(path), string(content)); err != nil {
		s.Render500(w, err)
	}
}

func (s *Server) Render404(w http.ResponseWriter, path string) {
	s.Log(ERROR, "File not found: %q", path)

	if err := s.Render(w, "File Not Found", "<h1>404: Page Not Found</h1>"); err != nil {
		s.Render500(w, err)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) Render500(w http.ResponseWriter, err error) {
	s.Log(ERROR, "Internal server error: %q", err)

	content := fmt.Sprintf("<h1>500 Internal Server Error: %s</h1>", err)

	if err := s.Render(w, "Internal Server Error", content); err != nil {
		s.Log(ERROR, "Failed to render page: %q", err)
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func (s *Server) Render(w http.ResponseWriter, file, content string) error {
	data := dict{"Port": s.Port, "File": file, "Content": content}
	return s.Renderer.Render(w, data)
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
		s.RenderFile(w, path.Join(uri[1:], "README.md"))
	case strings.HasSuffix(uri, ".md"):
		s.RenderFile(w, uri[1:])
	default:
		s.Render404(w, uri)
	}
	s.Log(INFO, "%s %s %s", r.Method, r.URL, w.Header().Get("Status"))
}

func (s *Server) Addr() string { return fmt.Sprintf("%s:%d", s.Host, s.Port) }

func (s *Server) Run() error {
	if err := os.Chdir(s.Directory); err != nil {
		return err
	}
	watcher, err := NewWatcher(s.Logger)
	if err != nil {
		return err
	}
	defer watcher.Close()
	livereload.Initialize()

	go watcher.Start(func(_ string) { livereload.ForceRefresh() })

	server := &http.Server{
		Addr:    s.Addr(),
		Handler: s,
	}

	s.Log(INFO, "Serving directory: %s on %s...", s.Directory, s.Addr())
	s.Log(DEBUG, "Debug mode enabled")
	return server.ListenAndServe()
}

func main() {
	var err error
	var debug bool

	server := &Server{Logger: NewStdLogger(std, INFO)}

	flag.StringVar(&server.Host, "h", "localhost", "Hostname from which the server will serve request")
	flag.IntVar(&server.Port, "p", 5678, "Port on which the server will serve request")
	flag.BoolVar(&debug, "d", false, "Debug mode, increase log level")
	flag.Parse()

	if debug {
		server.Logger = NewStdLogger(std, DEBUG)
	}
	if server.Directory = flag.Arg(0); server.Directory == "" {
		server.Directory = "."
	}
	if server.Renderer, err = NewMarkdownRenderer(); err != nil {
		server.Log(ERROR, "%s", err)
		return
	}
	server.Log(ERROR, "%s", server.Run())
}
