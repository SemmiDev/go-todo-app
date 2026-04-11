package docs

import "embed"

//go:embed swagger/*
var SwaggerFS embed.FS
