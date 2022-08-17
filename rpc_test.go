package rpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-qbit/rpc"
	mHello "github.com/go-qbit/rpc/internal/test/method/hello"
)

var (
	testRpc        *rpc.Rpc
	testHttpServer *httptest.Server
)

func init() {
	testRpc = rpc.New("github.com/go-qbit/rpc/internal/test/method")

	if err := testRpc.RegisterMethods(
		mHello.New(),
	); err != nil {
		panic(err)
	}

	testHttpServer = httptest.NewServer(testRpc)
}

func TestRpc_GetSwagger(t *testing.T) {
	swaggerJson := testRpc.GetSwagger(context.Background())

	_ = json.NewEncoder(os.Stderr).Encode(swaggerJson)

	if len(swaggerJson.Paths) == 0 {
		t.Fatalf("No registered paths")
	}
}

func TestRpc_ServeHTTP_Valid(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 100,
		StrParam: "test data",
		StructParam: mHello.StructV1{
			F1: 10,
		},
		StructPtrParam: &mHello.StructV1{
			F1: 20,
		},
	}))

	if status != 200 {
		t.Fatalf("Invalid status code = %d, expected 200. Data: '%s'", status, data)
	}

	var resp mHello.RespV1
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Data.Str != "test data" {
		t.Fatalf("Invalid Data.Str field = '%s', expected 'test data'", resp.Data.Str)
	}

	if resp.Data.Int != 100 {
		t.Fatalf("Invalid Data.Int field = %d, expected 100", resp.Data.Int)
	}
}

func TestRpc_ServeHTTP_InvalidJson(t *testing.T) {
	buf := bytes.NewBufferString(`a: 10`)
	status, data := doPost("/hello/v1", buf)
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func TestRpc_ServeHTTP_Validator_MinimumInt(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 10,
		StructParam: mHello.StructV1{
			F1: 10,
		},
	}))
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func TestRpc_ServeHTTP_Validator_MinimumUint(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 100,
		StructParam: mHello.StructV1{
			F1: 0,
		},
	}))
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func TestRpc_ServeHTTP_Validator_MaximumInt(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 1000,
		StructParam: mHello.StructV1{
			F1: 10,
		},
	}))
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func TestRpc_ServeHTTP_Validator_MaximumUint(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 1000,
		StructParam: mHello.StructV1{
			F1: 500,
		},
	}))
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func TestRpc_ServeHTTP_Validator_Pattern(t *testing.T) {
	status, data := doPost("/hello/v1", toJson(mHello.ReqV1{
		IntParam: 100,
		StrParam: "t",
		StructParam: mHello.StructV1{
			F1: 10,
		},
	}))
	if status != 400 {
		t.Fatalf("Invalid status code = %d, expected 400. Data: '%s'", status, data)
	}

	var resp rpc.Error
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatal(err)
	}

	if resp.Code != "INVALID_JSON" {
		t.Fatalf("Invalid error code field = '%s', expected 'INVALID_JSON'", resp.Code)
	}
}

func BenchmarkMethodDesc_Call(b *testing.B) {
	m := testRpc.GetMethod("/hello/v1")

	for i := 0; i < b.N; i++ {
		_, err := m.Call(context.Background(), bytes.NewBufferString(`{"int_param": 150, "str_param": "str value", "struct_param": {"f1": 10}}`))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func toJson(data interface{}) io.Reader {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		panic(err)
	}

	return buf
}

func doPost(method string, req io.Reader) (int, []byte) {
	resp, err := testHttpServer.Client().Post(testHttpServer.URL+method, "application/json", req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return resp.StatusCode, data
}
