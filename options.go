package tsid

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const (
	// SegmentLimit is the maximum number of segments
	SegmentsLimit = 63
	// EpochReservedDays indicates the minimum days approaching the end
	EpochReservedDays = 7
	// EpochMS is the default start timestamp,
	// measured in milliseconds starting
	// at midnight on December 12, 2022
	EpochMS = 1_670_774_400_000
	// The maximum width of the bits segment
	bitsMaxWidth = 63
)

// internal error string
const (
	errorSegmentMiss     = "required bits segments(Timestamp and Sequence)is missing"
	errorSegmentsTooMany = "bits segments too many"
	errorSegmentsEmpty   = "bits segments is empty"

	errorEpochTooSmall = "the EpochMS must be later than 1970-1-1T00:00:00"
	errorEpochTooLarge = "the EpochMS must be earlier than now"

	errorWidthInvalid  = "the width of bits segment is incorrect"
	errorWidthTooLarge = "the width of bits segment is too large"

	errorInvalidValue = "invalid value"

	errorDataSource  = "data source not provided"
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
	"Time.Minute",
	"Time.Second",
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
		return datetimeNames[int(d)]
	}
	return "Undefined"
}

const (
	// HostWidth is the default width of the bits segment,
	// value range [0, 63]
	HostWidth = 6
	// NodeWidth is the default width of the bits segment,
	// value range [0, 15]
	NodeWidth = 4
	// TimestampWidth is the default width of the bits segment.
	// It measures time by the number of seconds that have
	// elapsed since EpochMS, value range [0, 68 years after]
	TimestampWidth = 41
	// SequenceWidth is the default width of the bits segment,
	// value range [0, 4095]
	SequenceWidth = 12
)

type DataSource int

const (
	// Static indicates that the value is from default
	Static DataSource = iota
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
	// Random indicates that the value is from a random number
	RandomID
	// Provider indicates that the value is from data provider
	Provider
)

var dataSourceNames = []string{
	"Static", "Args", "OS", "Settings", "SequenceID", "DateTime", "RandomID", "Provider",
}

func (d DataSource) String() string {
	if int(d) < len(dataSourceNames) {
		return dataSourceNames[int(d)]
	}
	return "Undefined"
}

type DataProvider interface {
	Read(name string, index int) int64
}

type DataWrapper struct {
	data sync.Map
	// data map[string][]int64
	hook []func(string, int) int64
}

func DataWrap(hook ...func(string, int) int64) (dw *DataWrapper) {
	dw = &DataWrapper{
		hook: hook,
	}
	return
}

func (w *DataWrapper) Write(name string, data ...int64) *DataWrapper {
	w.data.Store(name, data)
	return w
}

func (w *DataWrapper) Read(name string, index int) int64 {
	if vs, f := w.data.Load(name); f {
		if v, o := vs.([]int64); o && index < len(v) {
			return v[index]
		}
	}
	for _, h := range w.hook {
		v := h(name, index)
		if v >= 0 {
			return v
		}
	}
	return -1
}

type Bits struct {
	// Source indicates that bits segment data source
	Source DataSource
	Width  byte
	Value  int64
	// Key indicates the data source key
	Key string
	// Index indicates the data source index
	Index int

	mask int64
}

// Host to make the bits segment of data center id, which value from settings
func Host(width byte, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    "Host",
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Node to make the bits segments of server node, which value from settings
func Node(width byte, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    "Node",
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Timestamp to make a bits segment, which value from system unix timestamp
func Timestamp(width byte, t DateTimeType) Bits {
	return Bits{
		Source: DateTime,
		Width:  width,
		Index:  int(t),
		// -1 ^ (-1 << (w % 64)),
	}
}

// Random to make a bits segment, which value from random number
func Random(width byte) Bits {
	return Bits{
		Source: RandomID,
		Width:  width,
		Index:  0,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Sequence to make a bits segment, which value from runtime sequence
func Sequence(width byte) Bits {
	return Bits{
		Source: SequenceID,
		Width:  width,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Fixed to make a bits segment, which value is fixed
func Fixed(width byte, value int64) Bits {
	return Bits{
		Source: Static,
		Width:  width,
		Value:  value,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Env to make a bits segment, which value from OS environment variable
func Env(width byte, name string, fallback int64) Bits {
	return Bits{
		Source: OS,
		Width:  width,
		Key:    name,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Arg to make a bits segment, which value from caller arguments
func Arg(width byte, index int, fallback int64) Bits {
	return Bits{
		Source: Args,
		Width:  width,
		Index:  index,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Option to make a bits segment, which value from settings in options
func Option(width byte, key string, fallback int64) Bits {
	return Bits{
		Source: Settings,
		Width:  width,
		Key:    key,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Data to make a bits segment, which value from data provider
func Data(width byte, key string, index int, fallback int64) Bits {
	return Bits{
		Source: Provider,
		Width:  width,
		Key:    key,
		Index:  index,
		Value:  fallback,
		// -1 ^ (-1 << (w % 64)),
	}
}

// Options MUST include DateTime segment AND SequenceID segment
type Options struct {
	// Min indicates the minimum days approaching the end
	Min,
	// EpochMS is the start timestamp
	EpochMS int64
	// Signed is used to on/off the sign bit
	Signed bool

	segments   []Bits
	settings   map[string]int64
	dataSource DataProvider
}

// Set to set the settings
func (o *Options) Set(k string, v int64) *Options {
	o.settings[k] = v
	return o
}

// Bind to bind the data provider
func (o *Options) Bind(p DataProvider) *Options {
	o.dataSource = p
	return o
}

// NewEpoch to set the start timestamp
func (o *Options) NewEpoch(v int64) *Options {
	o.EpochMS = v
	return o
}

// Add to add a bits segment
func (o *Options) Add(b Bits) *Options {
	o.segments = append(o.segments, b)
	return o
}

// O is a shortcut for make Options
func O(segments ...Bits) (o *Options) {
	o = &Options{
		segments: segments,
	}
	return o
}

// Define is a shortcut for make Options,
// segments MUST include DateTime segment AND SequenceID segment.
func Define(settings map[string]int64, segments ...Bits) (o *Options) {
	o = &Options{
		settings: settings,
		segments: segments,
	}
	return o
}

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

// Default is a shortcut for make Options,
// segments MUST include DateTime segment AND SequenceID segment.
func Default(host, node int64) *Options {
	return &Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: []Bits{
			Sequence(SequenceWidth),
			Node(NodeWidth, node),
			Host(HostWidth, host),
			Timestamp(TimestampWidth, TimestampMilliseconds),
		},
	}
}

// Shuffle is a shortcut for make Options.
func Shuffle(host, node int64) *Options {
	return &Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: []Bits{
			Sequence(SequenceWidth),
			Node(NodeWidth, host),
			Timestamp(31, TimestampSeconds),
			Host(HostWidth, node),
			Timestamp(10, TimeMillisecond),
		},
	}
}
