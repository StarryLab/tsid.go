package tsid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	// SegmentsLimit is the maximum number of segments
	SegmentsLimit = 63
	// EpochReservedDays indicates the minimum days approaching the end
	EpochReservedDays = 7
	// EpochMS is the default start timestamp,
	// measured in milliseconds starting
	// at midnight on December 12, 2022
	EpochMS = 1_670_774_400_000
	// The maximum width of the bit-segment
	bitsMaxWidth = 63
)

// internal error string
const (
	errorSegmentMiss     = "required bit-segments(Timestamp and Sequence)is missing"
	errorSegmentsTooMany = "bit-segments too many"
	errorSegmentsEmpty   = "bit-segments is empty"

	errorEpochTooSmall = "the EpochMS must be later than 1970-1-1T00:00:00"
	errorEpochTooLarge = "the EpochMS must be earlier than now"

	errorWidthInvalid  = "the width of bit-segment is incorrect"
	errorWidthTooLarge = "the width of bit-segment is too large"

	errorInvalidValue = "invalid value"

	errorInvalidType = "invalid data source type"

	errorTooPoor = "the end date has been reached and there are not enough identifiers"
	errorTooSlow = "the sequence width is too small and the time to generate identifiers is too slow"
)

type OptionsError struct {
	Name  string
	Extra []string
	Err   error
}

func (e *OptionsError) Error() string {
	ns := ""
	if len(e.Extra) > 0 {
		ns = "[" + strings.Join(e.Extra, ",") + "]"
	}
	return fmt.Sprintf(`tsid.Options: invalid options %s%s, reason: "%s"`,
		strconv.Quote(e.Name), ns, e.Err.Error())
}

func (e *OptionsError) Unwrap() error {
	return e.Err
}

func (e *OptionsError) SameAs(err error) bool {
	if errors.Is(err, e) {
		return true
	}
	if t, y := err.(*OptionsError); y {
		return t.Name == e.Name && t.Err.Error() == e.Err.Error()
	}
	return false
}

func invalidOption(name, reason string, extra ...string) *OptionsError {
	return &OptionsError{
		Name:  name,
		Extra: extra,
		Err:   errors.New(reason),
	}
}

type DateTimeType int

const (
	TimestampMilliseconds DateTimeType = iota
	TimestampNanoseconds
	TimestampMicroseconds
	TimestampSeconds
	TimeNanosecond
	TimeMicrosecond
	TimeMillisecond
	TimeSecond
	TimeMinute
	TimeHour
	TimeDay
	TimeMonth
	TimeYear
	TimeYearDay
	TimeWeekday
	TimeWeekNumber
)

var datetimeNames = []string{
	"Timestamp.Milliseconds",
	"Timestamp.NanoSeconds",
	"Timestamp.Microseconds",
	"Timestamp.Seconds",
	"Time.Nanosecond",
	"Time.Microsecond",
	"Time.Millisecond",
	"Time.Second",
	"Time.Minute",
	"Time.Hour",
	"Time.Day",
	"Time.Month",
	"Time.Year",
	"Time.YearDay",
	"Time.Weekday",
	"Time.WeekNumber",
}

func (d DateTimeType) String() string {
	if int(d) < len(datetimeNames) {
		return datetimeNames[d]
	}
	return "Undefined"
}

const (
	// HostWidth is the default width of the bit-segment,
	// value range [0, 63]
	HostWidth = 6
	// NodeWidth is the default width of the bit-segment,
	// value range [0, 15]
	NodeWidth = 4
	// TimestampWidth is the default width of the bit-segment.
	// It measures time by the number of seconds that have
	// elapsed since EpochMS, value range [0, 68 years after]
	TimestampWidth = 41
	// SequenceWidth is the default width of the bit-segment,
	// value range [0, 4095]
	SequenceWidth = 12
)

type DataSourceType int

const (
	// Static indicates that the value is from default
	Static DataSourceType = iota
	// Args indicates that the value is from arguments of caller
	Args
	// OS indicates that the value is from OS environment
	OS
	// Settings indicates that the value is from options
	Settings
	// SequenceID indicates that the value is from sequence value
	SequenceID
	// DateTime indicates that the value is from system unix timestamp in nanoseconds
	DateTime
	// RandomID indicates that the value is from a random number
	RandomID
	// Provider indicates that the value is from data provider
	Provider
)

var dataSourceTypeNames = []string{
	"Static",
	"Args",
	"OS",
	"Settings",
	"SequenceID",
	"DateTime",
	"RandomID",
	"Provider",
}

func (d DataSourceType) String() string {
	if int(d) < len(dataSourceTypeNames) {
		return dataSourceTypeNames[d]
	}
	return "Undefined"
}

type DataProvider interface {
	Read(query ...interface{}) (int64, error)
}

type Bits struct {
	// Source indicates that bit-segment data source
	Source DataSourceType
	Width  byte
	Value  int64
	// Key indicates the data source key
	Key string
	// Index indicates the data source index
	Index int

	mask  int64
	query []interface{}
}

// Host to make the bit-segment of data center id, which value from settings
func Host(width byte, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    "Host",
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Node to make the bit-segments of server node, which value from settings
func Node(width byte, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    "Node",
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Timestamp to make a bit-segment, which value from system unix timestamp
func Timestamp(width byte, t DateTimeType) Bits {
	return Bits{
		Source: DateTime,
		Width:  width,
		Index:  int(t),
		// -1 ^ (-1 << (w % 64)),
	}
}

// Random to make a bit-segment, which value from random number
func Random(width byte) Bits {
	return Bits{
		Source: RandomID,
		Width:  width,
		Index:  0,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Sequence to make a bit-segment, which value from runtime sequence
func Sequence(width byte) Bits {
	return Bits{
		Source: SequenceID,
		Width:  width,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Fixed to make a bit-segment, which value is fixed
func Fixed(width byte, value int64) Bits {
	return Bits{
		Source: Static,
		Width:  width,
		Value:  value,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Env to make a bit-segment, which value from OS environment variable
func Env(width byte, name string, fallback int64) Bits {
	return Bits{
		Source: OS,
		Width:  width,
		Key:    name,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Arg to make a bit-segment, which value from caller arguments
func Arg(width byte, index int, fallback int64) Bits {
	return Bits{
		Source: Args,
		Width:  width,
		Index:  index,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Option to make a bit-segment, which value from settings in options
func Option(width byte, key string, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    key,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Data to make a bit-segment, which value from data provider
func Data(width byte, source string, fallback int64, query ...interface{}) Bits {
	return Bits{
		Source: Provider,
		Width:  width,
		Key:    source,
		Index:  0,
		Value:  fallback,
		query:  query,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Options MUST include DateTime segment AND SequenceID segment
type Options struct {
	// ReservedDays indicates the minimum days approaching the end
	ReservedDays,
	// EpochMS is the start timestamp
	EpochMS int64
	// Signed is used to on/off the sign bit
	Signed bool

	segments []Bits
	settings map[string]int64
}

// Set to set the settings key and value
func (o *Options) Set(k string, v int64) *Options {
	if o.settings == nil {
		o.settings = map[string]int64{}
	}
	o.settings[k] = v
	return o
}

// NewEpoch to set the start timestamp
func (o *Options) NewEpoch(v int64) *Options {
	o.EpochMS = v
	return o
}

// Add to appends a bit-segment declaration
func (o *Options) Add(b Bits) *Options {
	w := b.Width
	b.mask = int64(-1 ^ (-1 << w))
	o.segments = append(o.segments, b)
	return o
}

// Patch is used to modify the settings of the bit field specified by w
func (o *Options) Patch(offset byte, key string, index int, fallback int64) *Options {
	if int(offset) < len(o.segments) {
		o.segments[offset].Key = key
		o.segments[offset].Index = index
		o.segments[offset].Value = fallback
	}
	return o
}

// O is a shortcut for make Options
func O(segments ...Bits) (o *Options) {
	return Segments(segments...)
}

// Segments is a shortcut for make Options
func Segments(segments ...Bits) (o *Options) {
	o = &Options{
		segments: segments,
	}
	return o
}

//
//// Define is a shortcut for make Options,
//// segments MUST include DateTime segment AND SequenceID segment.
//func Define(settings map[string]int64, segments ...Bits) (o *Options) {
//	o = &Options{
//		settings: settings,
//		segments: segments,
//	}
//	return o
//}

// Config is a shortcut for make Options,
// segments MUST include DateTime segment AND SequenceID segment.
func Config(host, node int64, segments ...Bits) *Options {
	return &Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: segments,
	}
}

const (
	// EnvServerHost is data center id, type: byte, value range [0, 31], 6 bits
	EnvServerHost = "SERVER_HOST_ID"
	// EnvServerNode is server node id, type: byte, value range [0, 15], 4 bits
	EnvServerNode = "SERVER_NODE_ID"
	// EnvDomainId is geo region id, type: int32, value range [0, 65535], 16 bits
	EnvDomainId = "SERVER_DOMAIN_ID"
	// EnvTimeEpoch is server epoch timestamp, type: int64, [0, 9_223_372_036_854_775_807]
	EnvTimeEpoch = "SERVER_EPOCH_TIMESTAMP"
)

var (
	predefined = map[string]*Options{
		"default": {
			EpochMS: EpochMS,
			segments: []Bits{
				Sequence(SequenceWidth),
				Env(NodeWidth, EnvServerNode, 0), // 4 bits [0, 15]
				Env(HostWidth, EnvServerHost, 0), // 6 bits [0, 31]
				Timestamp(TimestampWidth, TimestampMilliseconds),
			},
		},
		// 126 bits
		"random": {
			EpochMS: EpochMS,
			segments: []Bits{
				Random(63),
				Timestamp(31, TimestampSeconds),
				Env(NodeWidth, EnvServerNode, 0),
				Sequence(SequenceWidth),
				Env(HostWidth, EnvServerHost, 0),
				Timestamp(10, TimeMillisecond),
			},
		},
		"sequence": {
			segments: []Bits{
				Sequence(12),
				Timestamp(41, TimestampMilliseconds),
				Env(NodeWidth, EnvServerNode, 0),
				Env(HostWidth, EnvServerHost, 0),
			},
		},
		// 126 bits
		"openid": {
			segments: []Bits{
				Timestamp(31, TimestampSeconds),
				Env(4, EnvServerNode, 0),
				Sequence(14), // 14 bits [0, 16383]
				Env(6, EnvServerHost, 0),
				Timestamp(10, TimeMillisecond),
				Env(16, EnvDomainId, 0),
				Random(45),
			},
		},
		// 126 bits
		"test": {
			segments: []Bits{
				Timestamp(31, TimestampSeconds),    // 31 bits
				Fixed(4, 9),                        // 4 bits
				Env(10, EnvServerNode, 0),          // 10 bits
				Sequence(12),                       // 12 bits
				Data(5, "default", 3, "hit"),       // 5 bits
				Env(10, EnvServerHost, 0),          // 10 bits
				Data(5, "default", 9, "not_found"), // 5 bits
				Arg(8, 0, 0),                       // 8 bits
				Random(21),                         // 31 bits
				Option(10, "test", 0),              // 10 bits
				Timestamp(10, TimeMillisecond),     // 10 bits
			},
		},
		// TODO: auto-increment
	}
	aliases = map[string]string{
		"seqid":      "sequence",
		"sequenceid": "sequence",
		"classic":    "default",
		"snowflake":  "default",
		"shuffle":    "random",
		"testing":    "test",
		// TODO: auto-increment
		// "increment":      "sequence",
		// "auto-increment": "sequence",
	}
)

func init() {
	// reset EpochMS in all predefined options
	if s, f := os.LookupEnv(EnvTimeEpoch); f {
		if v, e := strconv.ParseInt(s, 10, 64); e == nil {
			for k := range predefined {
				predefined[k].EpochMS = v
			}
		}
	}
}

// Predefined obtains the predefined options specified by scope(case-insensitive),
// which includes "Default"(aliases: classic, snowflake), "Random"(aliases: shuffle),
// "OpenId", "SequenceId"(aliases: seq, seqid, increment, auto-increment),
// "Test"(aliases: testing) ... etc
func Predefined(scene string) (Options, bool) {
	scene = strings.ToLower(scene)
	if a, f := aliases[scene]; f {
		scene = a
	}
	if o, f := predefined[scene]; f {
		return *o, true
	}
	return Options{}, false
}

// Shuffle return predefined options "shuffle"(alias: random), 126 bits
func Shuffle() Options {
	return *predefined["random"]
}

// Default is a shortcut for make Options, which is the classic snowflake algorithm
func Default() Options {
	return *predefined["default"]
}

// OpenID is a shortcut for make Options, 126 bits
func OpenID() Options {
	return *predefined["openid"]
}

// SeqId is a shortcut for make Options
func SeqId() Options {
	return *predefined["sequence"]
}

// TODO: auto-increment
//// IncrementId is a shortcut for make Options
//func IncrementId() Options {
//	return *predefined["increment"]
//}

// Define adds the predefined options
func Define(scene string, options Options) bool {
	scene = strings.ToLower(scene)
	if _, f := aliases[scene]; f {
		return false
	}
	if _, f := predefined[scene]; f {
		return false
	}
	predefined[scene] = &options
	return true
}

func Play(count int) {
	if count <= 0 {
		count = 100
	}
	for n, o := range predefined {
		fmt.Printf("\n❤️ Options[%s]\n___________________________________________\n", n)
		b, e := New(*o)
		if e != nil {
			fmt.Printf("Options[%s] want: a builder instance, got error: %s\n", n, e)
			continue
		}
		for i := 0; i < count; i++ {
			id := b.Next()
			if id.IsZero() {
				fmt.Printf("Options[%s] want: valid ID, got zero\n", n)
				continue
			}
			buf := id.Bytes()
			m := binary.LittleEndian.Uint64(buf[:8])
			x := uint64(0)
			if len(buf) > 8 {
				x = binary.LittleEndian.Uint64(buf[8:])
			}
			if int64(m) != id.Main || int64(x) != id.Ext {
				fmt.Printf("ID.Bytes error: (%d, %d) != (%d, %d)\n", m, x, id.Main, id.Ext)
				return
			}
			fmt.Printf("%3d. %d %d %s\n", i+1, id.Ext, id.Main, id.String())
		}
	}
}
