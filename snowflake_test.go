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
	p := &ID{}
	c := 5000
	for i := 0; i < c; i++ {
		v := b.Next()
		// fmt.Printf("%3d. %+v\n", i+1, v)
		if p.Equal(v) {
			t.Error("invalid id, not auto-increment")
		}
		p = v
	}
}

func TestSimple(t *testing.T) {
	next, e := Simple(16)
	if e != nil {
		if e.Error() != "server time error" {
			t.Error("want: server time error, got: unknown error")
		}
		return
	}
	p := int64(0)
	c := 5000
	for i := 0; i < c; i++ {
		v := next()
		if v <= p {
			t.Error("error: invalid id, not auto-increment")
		}
		// fmt.Printf("%3d. %d %b\n", i+1, v, v)
		p = v
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
