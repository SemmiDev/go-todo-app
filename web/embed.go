// Package web handles the embedded static and template assets for the frontend.
package web

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed templates/*.html
var templateRaw embed.FS

//go:embed static/*
var staticRaw embed.FS

// TemplateFS provides access to all `.html` templates in the "templates" directory.
var TemplateFS fs.FS

// StaticFS provides access to static assets under "static" directory.
var StaticFS fs.FS

func init() {
	var err error
	TemplateFS, err = fs.Sub(templateRaw, "templates")
	if err != nil {
		log.Fatalf("web: sub templates: %v", err)
	}

	StaticFS, err = fs.Sub(staticRaw, "static")
	if err != nil {
		log.Fatalf("web: sub static: %v", err)
	}
}
