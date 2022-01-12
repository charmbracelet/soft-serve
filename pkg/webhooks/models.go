package webhooks

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/core/host"
	"io/ioutil"
	"net/http"
	"plugin"
)

type HTTPHandler func(http.ResponseWriter, *http.Request)

func (h HTTPHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	h(writer, request)
}

type Route struct {
	Path    string
	Method  string
	Handler http.Handler
}

type RouteSpec struct {
	Path        string `yaml:"path"`
	Method      string `yaml:"method"`
	HandlerName string `yaml:"handler-name"`
}

type Server struct {
	Routes             []Route
	TLSCertificatePath string
	TLSKeyPath         string
	Port               int
	Host               string
}

func NewServer(pluginPath, certPath, keyPath, host string, port int, specs ...RouteSpec) (Server, error) {
	p, err := plugin.Open(pluginPath)

	if err != nil {
		return Server{}, err
	}
	routes := []Route{}
	for _, spec := range specs {
		sym, err := p.Lookup(spec.HandlerName)
		if err != nil {
			return Server{}, err
		}
		handler, ok := sym.(HTTPHandler)

		if !ok {
			return Server{}, fmt.Errorf("%s is not of type http.Handler", spec.HandlerName)
		}
		routes = append(routes, Route{
			Path:    spec.Path,
			Method:  spec.Method,
			Handler: handler,
		})
	}
	return Server{
		Routes:             routes,
		TLSCertificatePath: certPath,
		TLSKeyPath:         keyPath,
	}, nil
}

func (s *Server) Serve(cs ...host.Configurator) error {
	app := iris.New()
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	for _, route := range s.Routes {
		app.Handle(route.Method, route.Path, func(ctx iris.Context) {
			w, r := ctx.ResponseWriter(), ctx.Request()
			route.Handler.ServeHTTP(w, r)
		})
	}

	if s.TLSCertificatePath != "" && s.TLSKeyPath != "" {
		tlsCert, err := ioutil.ReadFile(s.TLSCertificatePath)
		if err != nil {
			return err
		}

		tlsKey, err := ioutil.ReadFile(s.TLSKeyPath)
		if err != nil {
			return err
		}

		return iris.TLS(addr, string(tlsCert), string(tlsKey), cs...)(app)
	}

	return app.NewHost(&http.Server{Addr: addr}).Configure(cs...).ListenAndServe()
}

func (s *Server) ServeAsync(out chan<- error, cs ...host.Configurator) {
	go func() {
		out <- s.Serve(cs...)
	}()
}
