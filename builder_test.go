package tsid

import (
	"errors"
	"os"
	"testing"
	"time"
)

var ENV_TEST = "ENV_TEST"

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
	dp := &testDataSource{
		data: map[string]int64{
			"hit":   1,
			"other": 9,
		},
	}
	Register("my_data_source", dp)
}

func TestExts(t *testing.T) {
	os.Setenv(ENV_TEST, "1")
	defer os.Unsetenv(ENV_TEST)
	opt := ExtsOptions(10, 120)
	m, e := Make(opt)
	if e != nil {
		t.Fatal(e)
		return
	}
	m.ResetEpoch(0)
	for i := 0; i < 10; i++ {
		id, _ := m.Next(1, 2, 3, 4, 5, 6, 7, 8, 9)
		if id == nil {
			t.Fatal("builder config invalid")
			return
		}
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
				continue
			}
		}
	}
}

func BenchmarkExts(b *testing.B) {
	os.Setenv(ENV_TEST, "1")
	defer os.Unsetenv(ENV_TEST)
	opt := ExtsOptions(10, 9)
	m, e := Make(opt)
	if e != nil {
		b.Fatal(e)
		return
	}
	for i := 0; i < b.N; i++ {
		m.Next(1, 2, 3, 4, 5, 6, 7, 8, 9)
	}
}
func ExtsOptions(host, node int64) Options {
	return Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: []Bits{
			Host(16, host),                            // 16
			Timestamp(31, TimestampSeconds),           // 31
			Random(15),                                // 15
			Env(10, ENV_TEST, 0),                      // 10
			Fixed(5, 9),                               // 5
			Node(8, node),                             // 8
			Sequence(12),                              // 12
			Timestamp(10, TimeMillisecond),            // 10
			Arg(8, 0, 0),                              // 8
			Data(4, "my_data_source", 3, "hit"),       // 4
			Data(4, "my_data_source", 9, "not_found"), // 4
		},
	}
}

func TestOptionsError(t *testing.T) {
	now := time.Now().UnixNano() / nsPerMilliseconds
	h, n := int64(10), int64(10)
	tests := []struct {
		name string
		opt  *Options
		err  *OptionsError
	}{
		// {"EpochMS.TooSmall", Default(h, n).NewEpoch(-1), invalidOption("EpochMS", errorEpochTooSmall)},
		{"EpochMS.TooLarge", Default(h, n).NewEpoch(now + 5*msPerMinute), invalidOption("EpochMS", errorEpochTooLarge)},
		// {"EpochMS.TooPoor", Config(h, n).NewEpoch(now + 7*msPerDay), invalidOption("EpochMS", errorTooPoor)},
		{"Segments.Empty", Config(h, n), invalidOption("Segments", errorSegmentsEmpty)},
		{"Segments.Missing", Config(h, n,
			Host(6, 10),
			Node(4, 10),
			Timestamp(41, TimestampMilliseconds)),
			invalidOption("Segments", errorSegmentMiss)},
		{"Segments.Value", Config(h, n, Fixed(2, 10)), invalidOption("Segments", errorInvalidValue)},
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
	opt := Shuffle(0, 0).NewEpoch(now)
	if _, e := Make(*opt); e == nil {
		t.Fatal(`want: an error, got: an instance`)
		return
	} else if i, o := e.(*OptionsError); !o {
		t.Fatalf(`want: error(tsid.Options: invalid options "EpochMS", reason: "the end date has been reached and there are not enough identifiers"), got: error(%s)`, e)
		return
	} else if i.Name != "EpochMS" {
		t.Fatalf(`want: error(tsid.Options: invalid options "EpochMS", reason: "the end date has been reached and there are not enough identifiers"), got: error(%s)`, i)
		return
	}
	opt = Shuffle(0, 0)
	if m, e := Make(*opt); e != nil {
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
	for i := 0; i < 2000; i++ {
		id, _ := m.Next()
		no := en.Encode(id)
		de, _ := en.Decode(no)
		// t.Logf("\n%3d. ID: %d, De: %d, En: %s", i+1, id.Main, de.Main, no)
		if id.Main != de.Main {
			t.Errorf("decode error: next(%d), decode(%d)", id.Main, de.Main)
		}
		m.NextString()
	}
}
