package target

import "testing"

func TestFatalf(t *testing.T) {
	t.Fatalf("this should be ok")
}

func TestFatal(t *testing.T) {
	t.Fatal("this should be ok")
}

func BenchmarkFatal(b *testing.B) {
	b.Fatal("this should be ok")
}

func BenchmarkFatalf(b *testing.B) {
	b.Fatalf("this should be ok")
}

func FuzzFatal(f *testing.F) {
	f.Fatal("this should be ok")
}

func FuzzFatalf(f *testing.F) {
	f.Fatalf("this should be ok")
}
