package tsid

import (
	"testing"
	"time"
)

func TestBase64(t *testing.T) {
	e := Base64{Aligned: true}
	for i := 0; i < 100; i++ {
		n := &ID{
			Main: time.Now().UnixNano(),
			Ext:  Rand(byte(i)),
		}
		d := e.Encode(n)
		n2, e := e.Decode(d)
		if e != nil {
			t.Fatal("want: nothing, got: error ", e)
			return
		}
		if n.Main != n2.Main || n.Ext != n2.Ext {
			t.Fatal("want: [", n.Main, ", ", n.Ext, "] got: [", n2.Main, ",", n2.Ext, "]")
			return
		}
	}
}
func TestBase64Zero(t *testing.T) {
	e := Base64{Aligned: true}
	for i := int64(0); i < 100; i++ {
		e.Aligned = i%9 != 0
		n := &ID{
			Main: i % 10,
			Ext:  i % 5,
		}
		d := e.Encode(n)
		n2, e := e.Decode(d)
		if e != nil {
			t.Fatal("want: nothing, got: error ", e)
			return
		}
		if n.Main != n2.Main || n.Ext != n2.Ext {
			t.Fatal("want: [", n.Main, ", ", n.Ext, "] got: [", n2.Main, ",", n2.Ext, "]")
			return
		}
	}
}

func BenchmarkBase64EncodeMain(b *testing.B) {
	e := Base64{Aligned: true}
	for i := 0; i < b.N; i++ {
		n := &ID{
			Main: time.Now().UnixNano(),
		}
		e.Encode(n)
	}
}
func BenchmarkBase64EncodeExt(b *testing.B) {
	e := Base64{Aligned: true}
	for i := 0; i < b.N; i++ {
		n := &ID{
			Main: time.Now().UnixNano(),
			Ext:  time.Now().UnixNano(),
		}
		e.Encode(n)
	}
}
func BenchmarkBase64DecodeMain(b *testing.B) {
	e := Base64{Aligned: true}
	n := &ID{
		Main: time.Now().UnixNano(),
	}
	s := e.Encode(n)
	for i := 0; i < b.N; i++ {
		n2, e := e.Decode(s)
		if e != nil || n.Main != n2.Main || n.Ext != n2.Ext {
			b.Fatal("want: [", n.Main, ", ", n.Ext, "] got: [", n2.Main, ",", n2.Ext, "]")
			return
		}
	}
}
func BenchmarkBase64DecodeExt(b *testing.B) {
	e := Base64{Aligned: true}
	n := &ID{
		Main: time.Now().UnixNano(),
		Ext:  time.Now().UnixNano(),
	}
	s := e.Encode(n)
	for i := 0; i < b.N; i++ {
		n2, e := e.Decode(s)
		if e != nil || n.Main != n2.Main || n.Ext != n2.Ext {
			b.Fatal("want: [", n.Main, ", ", n.Ext, "] got: [", n2.Main, ",", n2.Ext, "]")
			return
		}
	}
}
