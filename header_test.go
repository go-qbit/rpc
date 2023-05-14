package rpc

import (
	"testing"
)

var (
	globalRes bool
	testCases = []struct {
		input string
		res   bool
	}{
		{"", false},
		{"gzip", true},
		{"test,gzip", true},
		{"gzip,test", true},
		{"test,gzip,test", true},
		{"testgzip", false},
		{"gziptest", false},
		{"testgziptest", false},
		{"test,gziptest", false},
		{"testgzip,test", false},
		{"test,testgzip,test", false},
	}
)

func TestCanGzip(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			if res := CanGzip(tc.input); res != tc.res {
				t.Fatalf("expected %v, got %v", tc.res, res)
			}
		})
	}
}

func TestCanGzipFast(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			if res := CanGzipFast(tc.input); res != tc.res {
				t.Fatalf("expected %v, got %v", tc.res, res)
			}
		})
	}
}

func BenchmarkCanGzip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			globalRes = CanGzip(tc.input)
		}
	}
}

func BenchmarkCanGzipFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			globalRes = CanGzipFast(tc.input)
		}
	}
}
