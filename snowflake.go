package tsid

import (
	"errors"
	"sync"
	"time"
)

// Snowflake implements the common snowflake algorithm.
// The value range of host is [0, 63].
// The value range of node is [0, 15].
func Snowflake(host, node int64) (*Builder, error) {
	opt := Options{
		EpochMS: EpochMS,
		segments: []Bits{
			Sequence(SequenceWidth),
			Node(NodeWidth, node), // 4 bits [0, 15]
			Host(HostWidth, host), // 6 bits [0, 31]
			Timestamp(TimestampWidth, TimestampMilliseconds),
		},
	}
	return Make(opt)
}

// Simple implements a classic snowflake algorithm(fixed width and position).
// The value range of server is [0, 1023].
//
//	if b, e := Simple(16); e == nil {
//	  fmt.Println("ID:")
//	  for i := 0; i < 100; i++ {
//	    fmt.Println(i+1, ". ", b())
//	  }
//	} else {
//	  fmt.Println("Error: ", e)
//	}
func Simple(server int64) (func() int64, error) {
	var b = struct {
		sync.Mutex
		start,
		now,
		sequence int64
	}{}
	seg := []struct {
		width byte
		shift byte
		mask  int64
	}{
		{12, 0, -1 ^ (-1 << 12)},  // sequence width 12 [0, 4095], low
		{10, 12, -1 ^ (-1 << 10)}, // server width 10 [0, 1023], middle
		{41, 22, -1 ^ (-1 << 41)}, // timestamp width 41, high
	}
	if server < 0 || server > seg[0].mask {
		return nil, errors.New("server id is too small or too large")
	}
	b.start = time.Now().UnixNano() / nsPerMilliseconds
	if b.start-EpochMS < 0 || b.start-EpochMS > seg[2].mask {
		return nil, errors.New("server time error")
	}
	return func() int64 {
		b.Lock()
		defer b.Unlock()
		sequence := int64(0)
		now := time.Now().UnixNano() / nsPerMilliseconds
		if b.now == now {
			sequence = (b.sequence + 1) & seg[0].mask
			if sequence == 0 {
				for b.now >= now {
					now = time.Now().UnixNano() / nsPerMilliseconds
				}
			}
		}
		b.sequence = sequence
		b.now = now
		// order by: 2 timestamp, 1 server, 0 sequence
		// MAYBE! The return value may be negative
		return ((now - EpochMS) << seg[2].shift) | (server << seg[1].shift) | b.sequence
	}, nil
}
