package hello

import (
	"context"
)

type Method struct {
}

func New() *Method {
	return &Method{}
}

func (m *Method) Caption(context.Context) string {
	return "Test"
}

func (m *Method) Description(context.Context) string {
	return "Test method"
}
