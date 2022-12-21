package server

import (
	"html/template"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/server/config"
)

func newHTTPServer(cfg *config.Config) *http.Server {
	r := http.NewServeMux()
	r.HandleFunc("/", repoIndexHandler(cfg))
	return &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.HTTP.Port)),
		Handler:           r,
		ReadHeaderTimeout: time.Second * 10,
		ReadTimeout:       time.Second * 10,
		WriteTimeout:      time.Second * 10,
		MaxHeaderBytes:    http.DefaultMaxHeaderBytes,
	}
}

var repoIndexHTMLTpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
 	<meta name="go-import" content="{{ .Config.HTTP.Domain }}/{{ .Repo }} git ssh://{{ .Config.Host }}:{{ .Config.SSH.Port }}/{{ .Repo }}">
</head>
</html>`))

func repoIndexHandler(cfg *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		repo := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")[0]
		log.Println("serving index for", repo)
		if err := repoIndexHTMLTpl.Execute(w, struct {
			Repo   string
			Config config.Config
		}{
			Repo:   repo,
			Config: *cfg,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
