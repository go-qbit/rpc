// Package openapi provides the minimum required part of the OpenAPI v3.0.3 data structure.
//
// The OpenApi struct can be serialized in JSON and YAML formats.
package openapi

type OpenApi struct {
	Openapi    string          `json:"openapi" yaml:"openapi"`
	Info       Info            `json:"info" yaml:"info"`
	Servers    []Server        `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]Path `json:"paths" yaml:"paths"`
	Components Components      `json:"components" yaml:"components"`
}

type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

type Server struct {
	Url         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Path struct {
	Post Operation `json:"post" yaml:"put"`
}

type Operation struct {
	Summary     string                  `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                  `json:"description,omitempty" yaml:"description,omitempty"`
	OperationId string                  `json:"operationId" yaml:"operationId"`
	Tags        []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
	RequestBody RequestBody             `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]ResponseBody `json:"responses" yaml:"responses"`
}

type RequestBody struct {
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool               `json:"required" yaml:"required"`
	Content     map[string]Content `json:"content,omitempty" yaml:"content,omitempty"`
}

type ResponseBody struct {
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]Content `json:"content,omitempty" yaml:"content,omitempty"`
}

type Content struct {
	Schema Schema `json:"schema" yaml:"schema"`
}

type Schema struct {
	Ref         string            `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string            `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string            `json:"format,omitempty" yaml:"format,omitempty"`
	Properties  map[string]Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema           `json:"items,omitempty" yaml:"items,omitempty"`
	Minimum     interface{}       `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     interface{}       `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	Pattern     string            `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas" yaml:"schemas"`
}
