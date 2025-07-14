package server

import (
	"html/template"
	"net/url"
	"strings"
	"time"
)

// createTemplateFunctions returns a map of template functions for use in HTML templates
func createTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return t.Format("2006-01-02 15:04:05")
		},
		"formatTimeAgo": func(t time.Time) string {
			if t.IsZero() {
				return "Never"
			}
			return time.Since(t).Round(time.Minute).String() + " ago"
		},
		"hasSuffix": func(s, suffix string) bool {
			return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
		},
		"urlEncode": func(s string) string {
			return url.QueryEscape(s)
		},
	}
}
