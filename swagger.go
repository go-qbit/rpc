package rpc

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-qbit/rpc/openapi"
)

func (r *Rpc) GetSwagger(ctx context.Context) *openapi.OpenApi {
	res := &openapi.OpenApi{
		Openapi: "3.0.3",
		Info: openapi.Info{
			Title: "GoRPC",
			Description: "The API is a mix of the REST and the JSONRPC ideas.\n\n" +
				"Each method has its own path.\n" +
				"The `POST` request with JSON data in body is used for transport.\n",
			Version: "1.0",
		},
		Paths: map[string]openapi.Path{},
		Components: openapi.Components{
			Schemas: map[string]openapi.Schema{},
		},
	}

	type errorDescription struct {
		Code        string
		Description string
	}

	for path, method := range r.methods {
		errors := make([]errorDescription, 0, len(method.Errors))
		for code, description := range method.Errors {
			errors = append(errors, errorDescription{code, description})
		}
		sort.Slice(errors, func(i, j int) bool {
			return errors[i].Code < errors[j].Code
		})

		errorsDescription := "### The business logic error\nPossible codes:\n"
		for _, e := range append([]errorDescription{
			{"INVALID_JSON", "Cannot parse JSON"},
		}, errors...) {
			errorsDescription += "* **" + e.Code + "**"
			if e.Description != "" {
				errorsDescription += ": " + e.Description
			}
			errorsDescription += "\n"
		}

		requestContentType := "application/json"
		t := method.Request.Elem()
		for i := 0; i < t.NumField(); i++ {
			if t.Field(i).Type == reflect.TypeOf((*File)(nil)).Elem() {
				requestContentType = "multipart/form-data"
				break
			}
		}

		res.Paths[path] = openapi.Path{
			Post: openapi.Operation{
				Summary:     method.Method.Caption(ctx),
				Description: method.Method.Description(ctx),
				OperationId: strings.Replace(path[1:], "/", "_", -1),
				Tags:        []string{"RPC methods"},
				RequestBody: openapi.RequestBody{
					Description: "",
					Required:    true,
					Content: map[string]openapi.Content{
						requestContentType: {
							Schema: r.getSchema(method.Request, res.Components.Schemas),
						},
					},
				},
				Responses: map[string]openapi.ResponseBody{
					"200": {
						Description: "### The result",
						Content: map[string]openapi.Content{
							"application/json": {
								Schema: r.getSchema(method.Response, res.Components.Schemas),
							},
						},
					},
					"400": {
						Description: errorsDescription,
						Content: map[string]openapi.Content{
							"application/json": {
								Schema: r.getSchema(reflect.TypeOf(Error{}), res.Components.Schemas),
							},
						},
					},
					"500": {
						Description: "### The internal server error",
					},
				},
			},
		}
	}

	return res
}

func (r *Rpc) getSchema(t reflect.Type, storage map[string]openapi.Schema) openapi.Schema {
	switch t.Kind() {
	case reflect.Ptr:
		return r.getSchema(t.Elem(), storage)

	case reflect.String:
		return openapi.Schema{
			Type: "string",
		}

	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
		return openapi.Schema{
			Type:   "integer",
			Format: "int64",
		}

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint8, reflect.Uint16:
		return openapi.Schema{
			Type:   "integer",
			Format: "int32",
		}

	case reflect.Bool:
		return openapi.Schema{Type: "boolean"}

	case reflect.Float32:
		return openapi.Schema{
			Type:   "number",
			Format: "float",
		}

	case reflect.Float64:
		return openapi.Schema{
			Type:   "number",
			Format: "double",
		}

	case reflect.Slice, reflect.Array:
		itemsSchema := r.getSchema(t.Elem(), storage)
		return openapi.Schema{
			Type:  "array",
			Items: &itemsSchema,
		}

	case reflect.Struct:
		name := r.typeName(t)

		if _, exists := storage[name]; !exists {
			jsonDataFields := make(map[string]openapi.Schema)
			fileFields := make(map[string]openapi.Schema)

			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				name := f.Name
				if !f.IsExported() {
					continue
				}
				jsonTag, ok := f.Tag.Lookup("json")
				if ok {
					name = strings.Split(jsonTag, ",")[0]
				}
				if name == "-" {
					continue
				}
				fieldSchema := r.getSchema(f.Type, storage)
				fieldSchema.Description = f.Tag.Get("desc")
				if err := addFieldRestrictions(f, &fieldSchema); err != nil {
					panic(fmt.Sprintf("Invalid validator value: %v", err))
				}

				if f.Type == reflect.TypeOf((*File)(nil)).Elem() {
					fileFields[name] = fieldSchema
					continue
				}
				jsonDataFields[name] = fieldSchema

			}

			schema := openapi.Schema{
				Type:       "object",
				Properties: map[string]openapi.Schema{},
			}

			if len(fileFields) > 0 {
				schema.Properties = fileFields
				schema.Properties["json_data"] = openapi.Schema{
					Type:       "object",
					Properties: jsonDataFields,
				}
			} else {
				schema.Properties = jsonDataFields
			}

			storage[name] = schema
		}

		return openapi.Schema{Ref: "#/components/schemas/" + name}

	case reflect.Interface, reflect.Map:
		if t == reflect.TypeOf((*File)(nil)).Elem() {
			return openapi.Schema{
				Type:   "string",
				Format: "binary",
			}
		}
		return openapi.Schema{Type: "object"}

	default:
		panic(fmt.Sprintf("Unknown type %s", t.String()))
	}
}

func (r *Rpc) typeName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := strings.TrimPrefix(t.PkgPath(), r.trimPrefix) + "/" + t.Name()
	name = strings.TrimPrefix(name, "/")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ToLower(name)

	return name
}

func addFieldRestrictions(f reflect.StructField, schema *openapi.Schema) error {
	for _, validator := range validators[f.Type.Kind()] {
		if err := validator.ToSwaggerSchema(f, schema); err != nil {
			return err
		}
	}

	return nil
}
