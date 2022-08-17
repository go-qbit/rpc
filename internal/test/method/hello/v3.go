package hello

import (
	"context"
	"io"

	"github.com/go-qbit/rpc"
)

type ReqV3 struct {
	IntParam int      `json:"int_param" desc:"An integer parameter"`
	Content  rpc.File `json:"content" desc:"Some file"`
}

type RespV3 struct {
	IntParam      int `json:"int_param"`
	ContentLength int `json:"content_length"`
}

func (m *Method) V3(ctx context.Context, r *ReqV3) (*RespV3, error) {
	data, err := io.ReadAll(r.Content)

	return &RespV3{IntParam: r.IntParam, ContentLength: len(data)}, err
}
