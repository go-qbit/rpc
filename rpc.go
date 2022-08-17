package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

var boundaryRe = regexp.MustCompile(`;.*boundary=(.*)`)

type Rpc struct {
	trimPrefix string
	methods    map[string]*MethodDesc
	options    *opts
}

type opts struct {
	cors      string
	maxMemory int64
}

type OptsFunc func(*opts)

func WithCors(allowedOrigins string) OptsFunc {
	return func(opts *opts) {
		opts.cors = allowedOrigins
	}
}

func WithMaxMemory(size int64) OptsFunc {
	return func(opts *opts) {
		opts.maxMemory = size
	}
}

func New(trimPrefix string, options ...OptsFunc) *Rpc {
	computedOpts := &opts{}
	for _, f := range options {
		f(computedOpts)
	}

	return &Rpc{
		trimPrefix: trimPrefix,
		methods:    map[string]*MethodDesc{},
		options:    computedOpts,
	}
}

func (r *Rpc) RegisterMethods(methods ...Method) error {
	for _, m := range methods {
		if err := r.RegisterMethod(m); err != nil {
			return err
		}
	}

	return nil
}

func (r *Rpc) RegisterMethod(method Method) error {
	mds, err := descsFromMethod(method, r.trimPrefix)
	if err != nil {
		return fmt.Errorf("cannot register mehtod %T: %w", method, err)
	}

	for _, md := range mds {
		r.methods[md.Path] = md
	}

	if err := bindErrors(method, r.trimPrefix, r.methods); err != nil {
		return fmt.Errorf("cannot bind errors for method %T: %w", method, err)
	}

	r.GetSwagger(context.Background()) // Check swagger

	return nil
}

func (r *Rpc) GetPaths() []string {
	res := make([]string, 0, len(r.methods))

	for path := range r.methods {
		res = append(res, path)
	}

	sort.Strings(res)

	return res
}

func (r *Rpc) GetMethod(path string) *MethodDesc {
	path = strings.TrimSuffix(path, "/")

	return r.methods[path]
}

func (r *Rpc) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if r.options.cors != "" {
		w.Header().Set("Access-Control-Allow-Origin", r.options.cors)
	}

	if request.Method == http.MethodOptions && r.options.cors != "" { // Cors
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "X-API-Key, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if request.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	defer request.Body.Close()

	method := r.GetMethod(request.URL.Path)
	if method == nil {
		http.NotFound(w, request)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	boundary := ""
	subs := boundaryRe.FindStringSubmatch(request.Header.Get("Content-Type"))
	if len(subs) > 0 {
		boundary = subs[1]
	}

	resp, err := method.Call(request.Context(), request.Body, boundary, r.options.maxMemory)
	if err != nil {
		if rpcErr, ok := err.(*Error); ok {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(rpcErr); err != nil {
				log.Printf("Cannot marshal error response: %v", err)
			}
			return
		}

		log.Printf("Cannot call %s: %v", method.Path, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Cannot marshal response: %v", err)
	}
}
