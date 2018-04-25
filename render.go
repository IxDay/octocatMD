package main

import (
	"io"
	"text/template"
)

type (
	Renderer interface {
		Render(w io.Writer, data dict) error
	}
	MarkdownRenderer struct {
		*template.Template
	}
)

func (mr *MarkdownRenderer) Render(w io.Writer, data dict) error {
	return mr.Execute(w, data)
}

func NewMarkdownRenderer() (*MarkdownRenderer, error) {
	data, err := Asset("data/main.html")
	if err != nil {
		return nil, err
	}
	return &MarkdownRenderer{
		template.Must(template.New("html").Parse(string(data))),
	}, nil
}
