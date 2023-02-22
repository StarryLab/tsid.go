package tsid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

var envTest = "ENV_TEST"

type testDataSource struct {
	data map[string]int64
}

func (d *testDataSource) Read(query ...interface{}) (int64, error) {
	if len(query) > 0 {
		if s, o := query[0].(string); o {
			if v, o := d.data[s]; o {
				return v, nil
			}
		}
	}
	return 0, errors.New("data not found")
}

func init() {
	// os.Setenv(EnvTimeEpoch, strconv.FormatInt(EpochMS, 10))
	dp := &testDataSource{
		data: map[string]int64{
			"hit":   1,
			"other": 9,
		},
	}
	Register("my_data_source", dp)
}

func TestExts(t *testing.T) {
	_ = os.Setenv(envTest, "1")
	defer func(key string) {
		_ = os.Unsetenv(key)
	}(envTest)
	opt, _ := Predefined("test")
	m, e := New(opt)
	if e != nil {
		t.Fatal(e)
		return
	}
	_ = m.ResetEpoch(0)
	for i := 0; i < 10; i++ {
		id := m.Next(1, 2, 3, 4, 5, 6, 7, 8, 9)
		if id == nil {
			t.Fatal("builder config invalid")
			return
		}
		_ = id.String()
		_ = id.Bytes()
		id.Signed = true
		_ = id.String()
	}
}

func TestDateTime(t *testing.T) {
	tt := []DateTimeType{
		TimeDay,
		TimeHour,
		TimeMillisecond,
		TimeMinute,
		TimeMonth,
		TimeSecond,
		TimeWeekNumber,
		TimeWeekday,
		TimeYear,
		TimeYearDay,
		99,
	}
	ts := []DateTimeType{
		TimestampMicroseconds,
		TimestampMilliseconds,
		TimestampNanoseconds,
		TimestampSeconds,
	}
	if TimeNanosecond.String() != "Time.Nanosecond" {
		t.Error("DateTimeType invalid")
	}
	if DateTimeType(100).String() != "Undefined" {
		t.Error("DateTimeType invalid")
	}
	for _, a := range tt {
		for _, b := range ts {
			opt := Options{
				segments: []Bits{
					Timestamp(31, b), // 31
					Sequence(12),     // 12
					Random(16),       // 16
					Timestamp(30, a), // 30
				},
			}
			if m, e := Make(opt); e == nil {
				m.Next()
			} else {
				t.Fatal(e)
				return
			}
		}
	}
}

func BenchmarkExts(b *testing.B) {
	_ = os.Setenv(envTest, "1")
	defer func(key string) {
		_ = os.Unsetenv(key)
	}(envTest)
	opt, _ := Predefined("test")
	m, e := Make(opt)
	if e != nil {
		b.Fatal(e)
		return
	}
	for i := 0; i < b.N; i++ {
		m.Next(1, 2, 3, 4, 5, 6, 7, 8, 9)
	}
}

func TestOptionsError(t *testing.T) {
	now := time.Now().UnixNano() / nsPerMilliseconds
	h, n := int64(10), int64(10)
	d := Default()
	d.NewEpoch(now + 5*msPerMinute)
	tests := []struct {
		name string
		opt  *Options
		err  *OptionsError
	}{
		// {"EpochMS.TooSmall", Default(h, n).NewEpoch(-1), invalidOption("EpochMS", errorEpochTooSmall)},
		{"EpochMS.TooLarge", &d,
			invalidOption("EpochMS", errorEpochTooLarge)},
		// {"EpochMS.TooPoor", Config(h, n).NewEpoch(now + 7*msPerDay), invalidOption("EpochMS", errorTooPoor)},
		{"Segments.Empty", Config(h, n),
			invalidOption("Segments", errorSegmentsEmpty)},
		{"Segments.Missing", Config(h, n,
			Host(6, 10),
			Node(4, 10),
			Timestamp(41, TimestampMilliseconds)),
			invalidOption("Segments", errorSegmentMiss)},
		{"Segments.Value", Config(h, n, Fixed(2, 10)),
			invalidOption("Segments", errorInvalidValue)},
		{"Segments.Type", Config(h, n, Bits{
			Source: 100,
		}), invalidOption("Segments", errorWidthInvalid)},
		{"Segments.Width", Config(h, n, Fixed(0, 0)),
			invalidOption("Segments", errorWidthInvalid)},
		{"Segments.Width.TooLarge", Config(h, n,
			Bits{Source: 0, Width: 20, Key: "First"},
			Bits{Source: 0, Width: 50, Key: "Second"},
			Bits{Source: 0, Width: 60, Key: "Error"}),
			invalidOption("Segments", errorWidthTooLarge)},
		{"Segments.Sequence.Width", Config(h, n,
			Host(6, 0),
			Node(4, 8),
			Timestamp(41, TimestampMilliseconds),
			Sequence(6),
		), invalidOption("Sequence.Width", errorTooSlow)},
		{"Segments.Required", Config(h, n,
			Host(6, 0),
			Node(4, 8),
			Sequence(10),
		), invalidOption("Segments", errorSegmentMiss)},
	}
	for _, o := range tests {
		t.Run(o.name, func(t *testing.T) {
			if _, e := Make(*o.opt); e == nil {
				t.Errorf("want: error(%s), no error occurs", o.err)
			} else if !o.err.SameAs(e) {
				t.Errorf("want: error(%s), got: error(%s)", o.err, e)
			}
		})
	}
}

func TestOptionsNewEpoch(t *testing.T) {
	now := time.Now().UnixNano() / nsPerMilliseconds
	opt := Shuffle()
	opt.NewEpoch(now)
	if _, e := Make(opt); e == nil {
		t.Fatal(`want: an error, got: an instance`)
		return
	} else if i, o := e.(*OptionsError); !o {
		t.Fatalf(`want: error(tsid.Options: invalid options "EpochMS", reason: "the end date has been reached and there are not enough identifiers"), got: error(%s)`, e)
		return
	} else if i.Name != "EpochMS" {
		t.Fatalf(`want: error(tsid.Options: invalid options "EpochMS", reason: "the end date has been reached and there are not enough identifiers"), got: error(%s)`, i)
		return
	}
	if m, e := Make(Shuffle()); e != nil {
		t.Errorf("want: builder instance, got: error(%s)", e)
	} else {
		e := m.ResetEpoch(now - 10*msPerDay)
		if e != nil {
			t.Errorf("want: successful, got: error(%s)", e)
		}
	}
}

func TestMake(t *testing.T) {
	en := Base64{Aligned: true}
	m, _ := Snowflake(10, 8)
	for i := 0; i < 20; i++ {
		id := m.Next()
		no := en.Encode(id)
		de, _ := en.Decode(no)
		// t.Logf("\n%3d. ID: %d, De: %d, En: %s", i+1, id.Main, de.Main, no)
		if id.Main != de.Main {
			t.Errorf("decode error: next(%d), decode(%d)", id.Main, de.Main)
		}
		m.NextString()
	}
	m.options.Add(Random(63)).Patch(1, "Node", 9, 4)
	for i := 0; i < 20; i++ {
		id := m.Next()
		no := en.Encode(id)
		de, _ := en.Decode(no)
		// t.Logf("\n%3d. ID: %d, De: %d, En: %s", i+1, id.Main, de.Main, no)
		if id.Main != de.Main {
			t.Errorf("decode error: next(%d), decode(%d)", id.Main, de.Main)
		}
		if id.Ext == 0 {
			t.Error("Options.Add failed")
		}
		m.NextString()
	}
}

func TestID(t *testing.T) {
	if DataSourceType(100).String() != "Undefined" {
		t.Error("DataSourceType.String invalid")
	}
	if Provider.String() != "Provider" {
		t.Error("DataSourceType.String invalid")
	}
	b, e := Make(Shuffle())
	if e != nil {
		t.Fatalf("want: a builder instance, got: error %s", e)
		return
	}
	id := b.Next()
	if !id.Equal(id) {
		t.Error("id.Equal not expected")
	}
	i2 := ID{
		Main:   id.Main,
		Ext:    id.Ext,
		Signed: id.Signed,
	}
	if !i2.Equal(id) {
		t.Error("id.Equal not expected")
	}
}

func TestSeqIDExt(t *testing.T) {
	Define("TestSeqIDExt",
		Options{
			segments: []Bits{
				Sequence(12),
				Timestamp(41, TimestampMilliseconds),
				Node(4, 4),
				Host(6, 8),
			},
		})
	opt, f := Predefined("TestSeqIDExt")
	if !f {
		t.Error("options define fail")
		return
	}
	opt.NewEpoch(EpochMS).Set("test", 99).Patch(2, "Node", 0, 5)
	if opt.EpochMS != EpochMS {
		t.Error("Options.NewEpoch failed")
	}
	if opt.settings["test"] != 99 {
		t.Error("Options.Set failed")
	}
	if opt.segments[2].Value != 5 {
		t.Error("Options.Patch failed")
	}

	if b, e := Make(opt); e == nil {
		b.Debug = true
		in := &DebugInfo{}
		var n int64
		for i := 0; i < 100000; i++ {
			d := b.NextInt64()
			if d < 1 {
				t.Error("an error ID(zero) was generated")
			}
			if d <= n {
				t.Error("the ID generated by SeqID are not incremental", in, b.info)
			}
			n = d
			in = b.info
		}
	} else {
		t.Error(e)
	}
}

func TestSeqID(t *testing.T) {
	o := SeqId()
	if c, e := New(o); e == nil {
		var n int64
		for i := 0; i < 100; i++ {
			d := c.NextInt64()
			if d < 1 {
				t.Error("an error ID(zero) was generated")
			}
			if d <= n {
				t.Error("the ID generated by SeqID are not incremental")
			}
			n = d
		}
	} else {
		t.Error(e)
	}
}

func BenchmarkSeqID(b *testing.B) {
	o := SeqId()
	c, e := New(o)
	if e != nil {
		b.Fatal(e)
		return
	}
	var n int64
	for i := 0; i < b.N; i++ {
		d := c.NextInt64()
		if d < 1 {
			b.Error("an error ID(zero) was generated")
		}
		if d <= n {
			b.Errorf("the ID generated by SeqID are not incremental, old: %d, new: %d", n, d)
		}
		n = d
	}
}

func TestAll(t *testing.T) {
	//_ = os.Setenv(EnvServerHost, "8")
	//_ = os.Setenv(EnvServerNode, "5")
	count := 10
	for n, o := range predefined {
		b, e := New(*o)
		if e != nil {
			t.Error("Predefined[", n, "]", " want: a builder instance, got error: ", e)
			continue
		}
		b.Debug = true
		for i := 0; i < count; i++ {
			id := b.Next()
			if id.IsZero() {
				t.Error("Predefined[", n, "]", " want: valid ID, got zero")
				continue
			}
			rs := fmt.Sprintf("%063b", id.Main)
			if id.Ext > 0 {
				rs = fmt.Sprintf("%063b", id.Ext) + rs
			}
			info := b.DebugInfo()
			cs := ""
			for j := len(b.options.segments); j > 0; j-- {
				w := b.options.segments[j-1].Width
				s := "%0" + strconv.FormatInt(int64(w), 10) + "b"
				ix := info.Raw[j-1]
				cs += fmt.Sprintf(s, ix)
			}
			if rs != cs {
				t.Errorf("want: %s, got: %s", cs, rs)
			}
			buf := id.Bytes()
			m := binary.LittleEndian.Uint64(buf[:8])
			x := uint64(0)
			if len(buf) > 8 {
				x = binary.LittleEndian.Uint64(buf[8:])
			}
			if int64(m) != id.Main || int64(x) != id.Ext {
				t.Errorf("ID.Bytes error: (%d, %d) != (%d, %d)", m, x, id.Main, id.Ext)
				return
			}
		}
	}
	Play(count)
}
