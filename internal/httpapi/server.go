package httpapi

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"rdmm404/voltr-finance/internal/api"
)

type Router struct {
	mux     *http.ServeMux
	paths   *http.ServeMux
	methods map[string]map[string]struct{}
}

func NewRouter() *Router {
	return &Router{mux: http.NewServeMux(), paths: http.NewServeMux(), methods: map[string]map[string]struct{}{}}
}

func (r *Router) Handle(method, path string, handler http.Handler) {
	method = strings.ToUpper(method)
	r.mux.Handle(method+" "+path, handler)
	shape := routeShape(path)
	if _, exists := r.methods[shape]; !exists {
		r.methods[shape] = map[string]struct{}{}
		r.paths.HandleFunc(shape, func(http.ResponseWriter, *http.Request) {})
	}
	r.methods[shape][method] = struct{}{}
}

// routeShape lets different methods use semantic wildcard names for the same
// path shape (for example, GET /categories/{code} and PATCH /categories/{id}).
func routeShape(path string) string {
	parts := strings.Split(path, "/")
	for index, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") && part != "{$}" {
			if strings.HasSuffix(part, "...}") {
				parts[index] = "{wildcard...}"
			} else {
				parts[index] = "{wildcard}"
			}
		}
	}
	return strings.Join(parts, "/")
}

func (r *Router) HandleFunc(method, path string, handler http.HandlerFunc) {
	r.Handle(method, path, handler)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	_, pattern := r.mux.Handler(request)
	if pattern != "" {
		r.mux.ServeHTTP(w, request)
		return
	}
	_, pathPattern := r.paths.Handler(request)
	if pathPattern == "" {
		WriteNotFound(w)
		return
	}
	methods := make([]string, 0, len(r.methods[pathPattern]))
	for method := range r.methods[pathPattern] {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	WriteMethodNotAllowed(w, strings.Join(methods, ", "))
}

type Config struct {
	Address           string
	APIKey            string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func (c *Config) setDefaults() {
	if c.Address == "" {
		c.Address = ":8080"
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = 5 * time.Second
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = 15 * time.Second
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = 30 * time.Second
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 60 * time.Second
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("API key is required")
	}
	if c.ReadHeaderTimeout < 0 || c.ReadTimeout < 0 || c.WriteTimeout < 0 || c.IdleTimeout < 0 {
		return errors.New("server timeouts cannot be negative")
	}
	return nil
}

type RegisterRoutes func(*Router)

func NewHandler(apiKey string, register RegisterRoutes) (http.Handler, error) {
	config := Config{APIKey: apiKey}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	apiRouter := NewRouter()
	if register != nil {
		register(apiRouter)
	}
	root := http.NewServeMux()
	root.HandleFunc("GET "+api.LivePath, func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	authenticated := BearerAPIKey(apiKey, apiRouter)
	root.Handle(api.APIPrefix, authenticated)
	root.Handle(api.APIPrefix+"/", authenticated)
	return root, nil
}

func NewServer(config Config, register RegisterRoutes) (*http.Server, error) {
	config.setDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	handler, err := NewHandler(config.APIKey, register)
	if err != nil {
		return nil, err
	}
	return &http.Server{Addr: config.Address, Handler: handler, ReadHeaderTimeout: config.ReadHeaderTimeout, ReadTimeout: config.ReadTimeout, WriteTimeout: config.WriteTimeout, IdleTimeout: config.IdleTimeout}, nil
}
