package tsid

// SeqID implements sequential identifiers.
// The value range of host is [0, 63].
// The value range of node is [0, 15].
//
//	if c, e := SeqID(10, 10); e == nil {
//	   fmt.Println("ID: ", c())
//	}
func SeqID(host, node int64) (func(args ...int64) int64, error) {
	opt := Options{
		segments: []Bits{
			Sequence(12),
			Timestamp(41, TimestampMilliseconds),
			Fixed(4, node),
			Fixed(6, host),
		},
	}
	b, e := Make(opt)
	if e != nil {
		return nil, e
	}
	return func(args ...int64) int64 {
		return b.NextInt64(args...)
	}, nil
}
