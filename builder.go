// TSID, A unique ID generator based on a timestamp or time series,
// inspired by Twitter's Snowflake.

package tsid

import (
	cr "crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	nsPerMilliseconds = 1_000_000
	usPerMilliseconds = 1_000
	msPerSecond       = 1000
	msPerMinute       = 60 * msPerSecond
	msPerHour         = 60 * msPerMinute
	msPerDay          = 24 * msPerHour
)

const (
	// Cutoff is the smallest number such that cutoff*64 > maxUint64
	cutoff           = 1 << 63
	uint64Max        = 1<<64 - 1
	uint63Max uint64 = 1<<63 - 1
)

type ID struct {
	Main,
	Ext int64
	Signed bool
}

type DebugInfo struct {
	Now,
	Sequence int64
	Bits []int64
}

type Builder struct {
	sync.Mutex

	Encoder Encoder
	Debug   bool

	ready   bool
	options *Options

	sequenceMask,
	now,
	sequence int64
}

func (b *Builder) tick() (t time.Time, now, sequence int64) {
	t = time.Now()
	now = t.UnixNano() / nsPerMilliseconds
	if now == b.now {
		sequence = (b.sequence + 1) & b.sequenceMask
		if sequence == 0 {
			for now <= b.now {
				t = time.Now()
				now = t.UnixNano() / nsPerMilliseconds
			}
		}
	}
	b.now = now
	b.sequence = sequence
	if b.options.EpochMS > 0 {
		now -= b.options.EpochMS
	}
	return
}

// rand generates a secure random number with a width specified by w,
// which is the expected bit width, value range is [1, 63].
func Rand(w byte) int64 {
	if w < 1 || w > 63 {
		return 0
	}
	c := w / 8
	if w%8 > 0 {
		c += 1
	}
	buf := make([]byte, c)
	n, e := cr.Read(buf)
	if e != nil || n < 1 {
		return 0
	}
	v := uint64(0)
	switch n {
	default:
		v = uint64(buf[0])
	case 2:
		v = uint64(buf[1])<<8 | uint64(buf[0])
	case 3:
		v = uint64(buf[2])<<16 | uint64(buf[1])<<8 | uint64(buf[0])
	case 4:
		v = uint64(buf[3])<<24 |
			uint64(buf[2])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[0])
	case 5:
		v = uint64(buf[4])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[2])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[0])
	case 6:
		v = uint64(buf[5])<<40 |
			uint64(buf[4])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[2])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[0])
	case 7:
		v = uint64(buf[6])<<48 |
			uint64(buf[5])<<40 |
			uint64(buf[4])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[2])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[0])
	case 8:
		v = uint64(buf[7])<<56 |
			uint64(buf[6])<<48 |
			uint64(buf[5])<<40 |
			uint64(buf[4])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[2])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[0])
	}
	m := -1 ^ (-1 << w)
	return int64(v & uint64(m))
}

func (b *Builder) Next(argv ...int64) (id *ID, info *DebugInfo) {
	if !b.ready {
		return nil, nil
	}
	b.Lock()
	defer b.Unlock()
	// ready
	var shift byte
	var overflow bool
	var main, ext int64
	var vs []int64
	tr, now, seq := b.tick()
	a := 0
	for _, segment := range b.options.segments {
		f := segment.Value
		key := segment.Key
		mask := segment.mask
		switch segment.Source {
		case Static:
		case Args:
			if a < len(argv) {
				f = argv[a]
			}
			a++
		case OS:
			if len(key) > 0 {
				if y, z := os.LookupEnv(key); z {
					if w, r := strconv.ParseInt(y, 10, 64); r != nil {
						f = w
					}
				}
			}
		case Settings:
			if len(key) > 0 {
				if y, z := b.options.settings[key]; z {
					f = y
				}
			}
		case SequenceID:
			f = seq
		case DateTime:
			switch DateTimeType(segment.Index) {
			case TimestampNanoseconds:
				f = now * nsPerMilliseconds
			case TimestampMicroseconds:
				f = now * usPerMilliseconds
			case TimestampMilliseconds:
				f = now
			case TimestampSeconds:
				f = now / 1_000
			case TimeMillisecond:
				f = now % 1000
			case TimeSecond:
				f = int64(tr.Second())
			case TimeMinute:
				f = int64(tr.Minute())
			case TimeHour:
				f = int64(tr.Hour())
			case TimeDay:
				f = int64(tr.Day())
			case TimeMonth:
				f = int64(tr.Month())
			case TimeYear:
				f = int64(tr.Year())
			case TimeYearDay:
				f = int64(tr.YearDay())
			case TimeWeekday:
				f = int64(tr.Weekday())
			case TimeWeekNumber:
				f = int64(tr.YearDay()/7 + 1)
			default:
				// TimestampMilliseconds
				f = now
			}
		case RandomID:
			f = Rand(segment.Width)
		case Provider:
			p := b.options.dataSource
			if p != nil {
				f = p.Read(segment.Key, segment.Index)
			}
		}
		if b.Debug {
			vs = append(vs, f)
		}
		if f < 0 {
			// TODO: negative
			f = 0
		}
		if f > mask {
			f &= mask
		}
		v := uint64(f)
		v2 := v
		if shift > 0 {
			v <<= shift
			if v >= uint63Max {
				v &= uint63Max
			}
		}
		if !overflow {
			main |= int64(v)
			sw := shift + segment.Width
			if sw > bitsMaxWidth {
				shift = sw - bitsMaxWidth
				ext = int64(v2 >> (segment.Width - shift))
			} else {
				shift += segment.Width
			}
			if sw >= bitsMaxWidth {
				overflow = true
			}
		} else {
			shift += segment.Width
			ext |= int64(v)
		}
	}
	id = &ID{
		Main:   main,
		Ext:    ext,
		Signed: b.options.Signed,
	}
	if b.Debug {
		info = &DebugInfo{
			Now:      now,
			Sequence: seq,
			Bits:     vs,
		}
	}
	return id, info
}

func (b *Builder) NextString(argv ...int64) string {
	i, _ := b.Next(argv...)
	e := b.Encoder
	if e == nil {
		s := strings.Builder{}
		s.Grow(27)
		if i.Signed {
			s.WriteByte('.')
		}
		if i.Ext > 0 {
			s.WriteString(strconv.FormatInt(i.Ext, 36))
		}
		if i.Main > 0 {
			m := strconv.FormatInt(i.Main, 36)
			s.WriteString(fmt.Sprintf("%013s", m))
		}
		return s.String()
	}
	return e.Encode(i)
}

func (b *Builder) ResetEpoch(epoch int64) error {
	if epoch < 0 {
		return invalidOption("EpochMS", errorEpochTooSmall)
	}
	now := time.Now().UnixNano() / nsPerMilliseconds
	if epoch > now {
		return invalidOption("EpochMS", errorEpochTooLarge)
	}
	min := int64(Min)
	if b.options.Min > min {
		min = b.options.Min
	}
	if now-epoch < min {
		return invalidOption("EpochMS", errorTooPoor)
	}
	b.options.EpochMS = epoch
	return nil
}

func Make(opt Options) (m *Builder, err error) {
	if opt.EpochMS <= 0 {
		if EpochMS < 0 {
			return nil, invalidOption("EpochMS", errorEpochTooSmall)
		}
		opt.EpochMS = EpochMS
	}
	now := time.Now().UnixNano() / nsPerMilliseconds
	if opt.EpochMS > now {
		return nil, invalidOption("EpochMS", errorEpochTooLarge)
	}
	min := Min
	if opt.Min > min {
		min = opt.Min
	}
	if now-opt.EpochMS < min {
		return nil, invalidOption("EpochMS", errorTooPoor)
	}
	t := byte(0)
	// Options MUST include DateTime segment AND SequenceID segment.
	required := map[DataSource]int{
		DateTime:   7,
		SequenceID: 0,
	}
	if len(opt.segments) < 1 {
		return nil, invalidOption("Segments", errorSegmentTooFew)
	}
	sequenceWidth := byte(0)
	for index, segment := range opt.segments {
		w := segment.Width
		if w < 1 || w > bitsMaxWidth {
			return nil, invalidOption("Segments", errorWidthInvalid)
		}
		if t+w > bitsMaxWidth*2 {
			return nil, invalidOption("Segments", errorWidthTooLarge)
		}
		t += w
		mask := int64(-1 ^ (-1 << w))
		opt.segments[index].mask = mask
		v := segment.Value
		switch segment.Source {
		case Static:
		case Args:
		case OS:
		case Settings:
		case SequenceID:
			if w > sequenceWidth {
				sequenceWidth = segment.Width
			}
			delete(required, SequenceID)
			v = 0
		case RandomID:
			v = 0
		case DateTime:
			switch segment.Index {
			case int(TimestampNanoseconds),
				int(TimestampMicroseconds),
				int(TimestampMilliseconds),
				int(TimestampSeconds):
				delete(required, DateTime)
			}
			v = 0
		case Provider:
			p := opt.dataSource
			if p == nil {
				return nil, invalidOption("Segments", errorDataSource)
			}
		default:
			return nil, invalidOption("Segments", errorInvalidType)
		}
		if v > mask {
			return nil, invalidOption("Segments", errorInvalidValue)
		}
	}
	if len(required) > 0 {
		return nil, invalidOption("Segments", errorSegmentMiss)
	}
	if sequenceWidth < 8 {
		return nil, invalidOption("Sequence.Width", errorTooSlow)
	}
	m = &Builder{
		options:      &opt,
		sequenceMask: -1 ^ (-1 << sequenceWidth),
		ready:        true,
	}
	return
}
