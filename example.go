package tsid

// SeqID implements sequential identifiers.
// The value range of host is [0, 63].
// The value range of node is [0, 15].
//
//  if c, e := SeqID(10, 10); e == nil {
//     fmt.Println("ID: ", c())
//  }
func SeqID(host, node int64) (func(args ...int64) int64, error) {
	opt := Options{
		settings: map[string]int64{
			"Host": host,
			"Node": node,
		},
		segments: []Bits{
			Sequence(SequenceWidth),
			Timestamp(41, TimestampMilliseconds),
			Node(NodeWidth, node),
			Host(HostWidth, host),
		},
	}

	b, e := Make(opt)
	if e != nil {
		return nil, e
	}
	return func(args ...int64) int64 {
		i, _ := b.Next(args...)
		return i.Main
	}, nil
}
