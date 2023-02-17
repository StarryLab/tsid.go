package tsid

import (
	"testing"
)

func TestSnowflake(t *testing.T) {
	r := invalidOption("Segments", errorInvalidValue)
	_, e := Snowflake(0, 16)
	if e != nil {
		if x, f := e.(*OptionsError); f {
			if x.Name != r.Name || x.Error() != r.Error() {
				t.Fatalf("want: %s, got: %s", r, e)
				return
			}
		} else {
			t.Fatalf("want: OptionsError, got: %s", e)
			return
		}
	} else {
		t.Fatal("want: error, got: an instance")
		return
	}
	b, e := Snowflake(0, 15)
	if e != nil {
		t.Fatalf("want: an instance, got: error(%s)", e)
		return
	}
	for i := 0; i < 10; i++ {
		b.Next()
	}
}

func TestSimple(t *testing.T) {
	c, e := Simple(16)
	if e != nil {
		if e.Error() != "server time error" {
			t.Error("want: server time error, got: unknown error")
		}
		return
	}
	for i := 0; i < 10; i++ {
		c()
	}
}

func BenchmarkSnowflake(b *testing.B) {
	c, e := Snowflake(10, 15)
	if e != nil {
		b.Fatal("error", e)
		return
	}
	for i := 0; i < b.N; i++ {
		c.Next()
	}
}

func BenchmarkSimple(b *testing.B) {
	c, e := Simple(168)
	if e != nil {
		b.Fatal("error", e)
		return
	}
	for i := 0; i < b.N; i++ {
		c()
	}
}
