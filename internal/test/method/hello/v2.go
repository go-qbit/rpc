package hello

import (
	"context"
)

type ReqV2 struct {
	IntParam int `json:"int_param" desc:"An integer parameter"`
}

func (m *Method) V2(ctx context.Context, r *ReqV2) (int, error) {
	return r.IntParam, nil
}
