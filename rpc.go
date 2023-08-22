package rpc

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/go-qbit/rpc/htb"
)

var boundaryRe = regexp.MustCompile(`;.*boundary=(.*)`)

type Rpc struct {
	trimPrefix string
	methods    map[string]*MethodDesc
	options    opts
}

type opts struct {
	cors      *cors
	maxMemory int64
}

type cors struct {
	allowOrigin  string
	allowHeaders string
	allowMethods string
	maxAge       string
}

type OptsFunc func(*opts)

func WithCors(allowedOrigins string) OptsFunc {
	return func(opts *opts) {
		WithCorsV2(
			strings.Split(allowedOrigins, ", "),
			[]string{"X-API-Key", "Content-Type"},
			[]string{"POST", "OPTIONS"},
			"86400",
		)
	}
}

func WithCorsV2(origin, headers, methods []string, maxAge string) OptsFunc {
	return func(opts *opts) {
		c := &cors{}
		if origin != nil {
			c.allowOrigin = strings.Join(origin, ", ")
		}

		if headers != nil {
			c.allowHeaders = strings.Join(headers, ", ")
		}

		if methods != nil {
			c.allowMethods = strings.Join(methods, ", ")
		}

		if maxAge != "" {
			c.maxAge = maxAge
		}

		opts.cors = c
	}
}

func WithMaxMemory(size int64) OptsFunc {
	return func(opts *opts) {
		opts.maxMemory = size
	}
}

func New(trimPrefix string, options ...OptsFunc) *Rpc {
	computedOpts := opts{}
	for _, f := range options {
		f(&computedOpts)
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
	if r.options.cors != nil {
		w.Header().Set("Access-Control-Allow-Origin", r.options.cors.allowOrigin)
		w.Header().Set("Access-Control-Allow-Headers", r.options.cors.allowHeaders)
		w.Header().Set("Access-Control-Allow-Methods", r.options.cors.allowMethods)
		w.Header().Set("Access-Control-Max-Age", r.options.cors.maxAge)
	}

	if request.Method == http.MethodOptions && r.options.cors.allowOrigin != "" { // Cors
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

	var writer io.Writer = w
	if CanGzipFast(request.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")

		gzW := gzip.NewWriter(writer)
		gzW.Header.Name = htb.RandomString() // See https://ieeexplore.ieee.org/document/9754554

		defer gzW.Close()

		writer = gzW
	}

	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		log.Printf("Cannot marshal response: %v", err)
	}
}
