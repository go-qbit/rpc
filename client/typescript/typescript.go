package typescript

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-qbit/rpc"
)

func New(rpc *rpc.Rpc, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		types := map[string]reflect.Type{}

		methodsCode := &bytes.Buffer{}

		for _, path := range rpc.GetPaths() {
			m := rpc.GetMethod(path)

			methodNameParts := strings.Split(strings.TrimPrefix(path, "/"), "/")
			for i, s := range methodNameParts {
				methodNameParts[i] = strings.ToUpper(s[:1]) + s[1:]
			}
			methodName := strings.Join(methodNameParts, "")

			methodsCode.WriteString("\n\n  // ")
			methodsCode.WriteString(m.Method.Description(r.Context()))
			methodsCode.WriteString("\n")
			methodsCode.WriteString(`  public static `)
			methodsCode.WriteString(methodName)
			methodsCode.WriteString("(request: ")
			methodsCode.WriteString(toTsTypeName(m.Request, prefix))
			methodsCode.WriteString("): Promise<")
			methodsCode.WriteString(toTsTypeName(m.Response, prefix))
			methodsCode.WriteString("> {\n    return this.post('")
			methodsCode.WriteString(path)
			methodsCode.WriteString("', request,'")
			methodsCode.WriteString(checkContentType(m.Request))
			methodsCode.WriteString("') as Promise<")
			methodsCode.WriteString(toTsTypeName(m.Response, prefix))
			methodsCode.WriteString(">\n  }")

			addTsStructTypes(m.Request, prefix, types)
			addTsStructTypes(m.Response, prefix, types)
		}

		methodsCode.WriteString("\n}")

		typesNames := make([]string, 0, len(types))
		for t := range types {
			typesNames = append(typesNames, t)
		}
		sort.Strings(typesNames)

		for _, name := range typesNames {
			_, _ = io.WriteString(w, "export type ")
			_, _ = io.WriteString(w, name)
			_, _ = io.WriteString(w, " = {")

			for i := 0; i < types[name].NumField(); i++ {
				field := types[name].Field(i)
				name := field.Tag.Get("json")
				name = strings.Split(name, ",")[0]
				if name == "" {
					name = field.Name
				}
				if name == "-" {
					continue
				}

				_, _ = io.WriteString(w, "\n  ")
				_, _ = io.WriteString(w, name)
				if field.Type.Kind() == reflect.Ptr {
					_, _ = io.WriteString(w, "?")
				}
				_, _ = io.WriteString(w, ": ")
				_, _ = io.WriteString(w, toTsTypeName(field.Type, prefix))

				if description := field.Tag.Get("desc"); description != "" {
					_, _ = io.WriteString(w, "  // ")
					_, _ = io.WriteString(w, description)
				}
			}

			_, _ = io.WriteString(w, "\n}\n\n")
		}

		_, _ = io.WriteString(w, tsLibBody)
		_, _ = methodsCode.WriteTo(w)
	}
}

func toTsTypeName(varType reflect.Type, prefix string) string {
	if override := typesOverrides[varType.PkgPath()+"."+varType.Name()]; override != "" {
		return override
	}

	typeParts := strings.Split(strings.TrimPrefix(varType.PkgPath(), prefix), "/")
	for i, part := range typeParts {
		typeParts[i] = strings.Title(part)
	}
	typePrefix := strings.Join(typeParts, "")

	switch varType.Kind() {
	case reflect.Slice:
		return toTsTypeName(varType.Elem(), prefix) + "[]"
	case reflect.Struct:
		sName := varType.Name()
		if sName == "" {
			sName = "Struct_" + strconv.FormatUint(uint64(crc32.ChecksumIEEE([]byte(varType.String()))), 16)
		}
		return typePrefix + strings.Title(sName)
	case reflect.Map:
		return "Record<" + toTsTypeName(varType.Key(), prefix) + ", " + toTsTypeName(varType.Elem(), prefix) + ">"
	case reflect.Ptr:
		return toTsTypeName(varType.Elem(), prefix)
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Float32, reflect.Float64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	case reflect.Interface:
		if varType == reflect.TypeOf((*rpc.File)(nil)).Elem() {
			return "File"
		} else {
			return "unknown"
		}
	default:
		panic(fmt.Sprintf("Unknown kind %s", varType.Kind().String()))
	}
}

func addTsStructTypes(st reflect.Type, prefix string, m map[string]reflect.Type) {
	if typesOverrides[st.PkgPath()+"."+st.Name()] != "" {
		return
	}

	switch st.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Array:
		addTsStructTypes(st.Elem(), prefix, m)

	case reflect.Struct:
		m[toTsTypeName(st, prefix)] = st
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			if field.Tag.Get("json") != "-" {
				addTsStructTypes(field.Type, prefix, m)
			}
		}

	case reflect.Map:
		addTsStructTypes(st.Elem(), prefix, m)

	case reflect.String, reflect.Bool, reflect.Float32, reflect.Float64, reflect.Interface,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return

	default:
		panic(fmt.Sprintf("Unknown kind %s", st.Kind().String()))
	}
}

var (
	typesOverrides = map[string]string{
		"time.Time": "string",
	}

	tsLibBody = `export class ApiError extends Error {
  private readonly _code: string
  private readonly _message: string
  private readonly _data: unknown

  constructor(code: string, message: string, data: unknown) {
    super(message)
    this._code = code
    this._message = message
    this._data = data
  }

  get code(): string {
    return this._code
  }

  get message(): string {
    return this._message
  }

  get data(): unknown {
    return this._data
  }
}

export default class API {
  static url = '/api'
  static customHeaders: () => Promise<Record<string, string>> | undefined

  private static requestToFormData (request: any): FormData{
    const form = new FormData()
    const json_data:any = {}
    for (let name in request){
      if (request[name] instanceof File){
        form.append(name, request[name])
        continue
      }
        json_data[name] = request[name]
    }
    if (Object.keys(json_data).length!==0) form.append("json_data", JSON.stringify(json_data))
    return form
  }

  private static async post(method: string, request: unknown, contentType: string): Promise<unknown> {
    return fetch(
      this.url + method,
      {
		method: 'post',
        headers: Object.assign(this.customHeaders? await this.customHeaders()!: {},
        contentType === 'application/json'? {'Content-Type': contentType}: {}
        ),
        body: contentType === 'application/json'? JSON.stringify(request) : this.requestToFormData(request)  
      }
    )
      .then(response => {
        return new Promise<Response>((resolve, reject) => {
          switch (response.status) {
            case 200:
              resolve(response)
              break
            case 400:
              response.json().then(err => {
                reject(new ApiError(err.code, err.message, err.data))
              })
              break
            default:
              response.text().then(text => {
                reject(new Error(text || response.statusText))
              })
          }
        })
      })
      .then((response) => response.json())
  }`
)

func checkContentType(st reflect.Type) string {
	st = st.Elem()
	ret := "application/json"
	for i := 0; i < st.NumField(); i++ {
		if st.Field(i).Type == reflect.TypeOf((*rpc.File)(nil)).Elem() {
			ret = "multipart/form-data"
			break
		}
	}

	return ret
}
