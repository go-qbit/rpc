package hello

import (
	"context"
)

type ReqV4 struct {
}

type RecursiveStruct struct {
	F        string                      `json:"f"`
	Child    *RecursiveStruct            `json:"child"`
	Children []*RecursiveStruct          `json:"children"`
	Map      map[string]*RecursiveStruct `json:"map"`
}
type RespV4 struct {
	Rs *RecursiveStruct `json:"rs"`
}

func (m *Method) V4(ctx context.Context, r *ReqV3) (*RespV4, error) {
	return &RespV4{
		Rs: &RecursiveStruct{
			F:     "Parent",
			Child: &RecursiveStruct{F: "Child"},
		},
	}, nil
}
