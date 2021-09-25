package hello

import (
	"context"

	"github.com/go-qbit/rpc"
)

type ReqV1 struct {
	IntParam       int       `json:"int_param" desc:"An integer parameter" minimum:"100" maximum:"200"`
	StrParam       string    `json:"str_param" desc:"A string parameter" pattern:".{2,}"`
	ArrParam       []string  `json:"arr_param" desc:"An array parameter"`
	StructParam    StructV1  `json:"struct_param"`
	StructPtrParam *StructV1 `json:"struct_ptr_param"`
	WithErr        bool      `json:"with_err"`
}

type StructV1 struct {
	F1 uint `json:"f1" minimum:"1" maximum:"200"`
}

type RespV1 struct {
	Message string `json:"message"  desc:"Just a message"`
	Data    DataV1 `json:"data"`
}

type DataV1 struct {
	Int int    `json:"int,omitempty"`
	Str string `json:"str"`
}

var ErrorsV1 struct {
	Error1 rpc.ErrorFunc `desc:"Error 1"`
	Error2 rpc.ErrorFunc `desc:"Error 2"`
	Error3 rpc.ErrorFunc `desc:"Error 3"`
}

func (m *Method) ErrorsV1() interface{} {
	return &ErrorsV1
}

func (m *Method) V1(ctx context.Context, r *ReqV1) (*RespV1, error) {
	if r.WithErr {
		return nil, ErrorsV1.Error1("test")
	}

	return &RespV1{
		Message: "Hello, world",
		Data: DataV1{
			Int: r.IntParam,
			Str: r.StrParam,
		},
	}, nil
}
