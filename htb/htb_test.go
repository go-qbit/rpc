package htb

import (
	"testing"
)

func TestRandomString(t *testing.T) {
	for n := 0; n < 100; n++ {
		str := RandomString()

		switch {
		case len(str) != paddingSize:
			t.Fatalf("Expected all results from RandomString to have length %d, got %d", paddingSize, len(str))
		}
	}
}

func BenchmarkRandomString(b *testing.B) {
	for n := 0; n < b.N; n++ {
		RandomString()
	}
}
