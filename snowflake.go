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
	opt := *Default(host, node)
	return Make(opt)
}

// Simple implements a classic snowflake algorithm(fixed width and position).
// The value range of server is [0, 1023].
//   if b, e := Simple(16); e == nil {
//     fmt.Println("ID:")
//     for i := 0; i < 100; i++ {
//       fmt.Println(i+1, ". ", b())
//     }
//   } else {
//     fmt.Println("Error: ", e)
//   }
func Simple(server int64) (func() int64, error) {
	var b = struct {
		sync.Mutex
		now,
		sequence int64
	}{}
	s := []struct {
		width byte
		shift byte
		mask  int64
	}{
		{12, 0, -1 ^ (-1 << 12)},  // sequence width 12 [0, 4095], low
		{10, 12, -1 ^ (-1 << 10)}, // server width 10 [0, 1023], middle
		{41, 22, -1 ^ (-1 << 41)}, // timestamp width 41, high
	}
	if server < 0 || server > s[1].mask {
		return nil, errors.New("server id is too small or too large")
	}
	now := time.Now().UnixNano() / nsPerMilliseconds
	t := now - EpochMS
	if t < 0 || t > s[2].mask {
		return nil, errors.New("server time error")
	}
	return func() int64 {
		b.Lock()
		defer b.Unlock()
		b.sequence = 0
		now := time.Now().UnixNano() / nsPerMilliseconds
		if b.now == now {
			b.sequence = (b.sequence + 1) & s[2].mask
			if b.sequence == 0 {
				for b.now >= now {
					now = time.Now().UnixNano() / nsPerMilliseconds
				}
			}
		}
		t := now - EpochMS
		b.now = now
		// order by: 2 timestamp, 1 server, 0 sequence
		// MAYBE! The return value may be negative
		return (t << s[2].shift) | (server << s[1].shift) | b.sequence
	}, nil
}
