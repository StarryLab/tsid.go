package tsid

import (
	"testing"
	"time"
)

func TestExts(t *testing.T) {
	opt := *ExtsOptions(1000, 120)
	m, e := Make(opt)
	if e != nil {
		t.Fatal(e)
		return
	}
	for i := 0; i < 1; i++ {
		id, _ := m.Next()
		if id == nil {
			t.Fatal("builder config invalid")
			return
		}
	}
}

func BenchmarkExts(b *testing.B) {
	opt := *ExtsOptions(100, 9)
	m, e := Make(opt)
	if e != nil {
		b.Fatal(e)
		return
	}
	for i := 0; i < b.N; i++ {
		m.Next()
	}
}
func ExtsOptions(host, node int64) *Options {
	return &Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: []Bits{
			Host(20, host),                  // 20
			Timestamp(31, TimestampSeconds), // 31
			Random(31),                      // 31
			Fixed(15, 9),                    // 15
			Node(7, node),                   // 7
			Sequence(12),                    // 12
			Timestamp(10, TimeMillisecond),  // 10
		},
	}
}

func TestOptionsError(t *testing.T) {
	now := time.Now().UnixNano() / nsPerMilliseconds
	h, n := int64(10), int64(10)
	tests := []struct {
		name string
		opt  *Options
		err  error
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
			} else if e.Error() != o.err.Error() {
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
	for i := 0; i < 10; i++ {
		id, _ := m.Next()
		no := en.Encode(id)
		de, _ := en.Decode(no)
		// t.Logf("\n%3d. ID: %d, De: %d, En: %s", i+1, id.Main, de.Main, no)
		if id.Main != de.Main {
			t.Errorf("decode error: next(%d), decode(%d)", id.Main, de.Main)
		}
	}
}
