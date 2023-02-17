
# TSID

![GitHub](https://img.shields.io/github/license/StarryLab/tsid.go) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/StarryLab/tsid.go) [![Go Reference](https://pkg.go.dev/badge/github.com/StarryLab/tsid.go@v1.0.0-alpha.svg)](https://pkg.go.dev/github.com/StarryLab/tsid.go@v1.0.0-alpha) ![GitHub release (latest SemVer including pre-releases)](https://img.shields.io/github/v/release/StarryLab/tsid.go?include_prereleases&sort=semver) ![GitHub Repo stars](https://img.shields.io/github/stars/StarryLab/tsid.go?style=social)
![GitHub last commit](https://img.shields.io/github/last-commit/StarryLab/tsid.go) ![GitHub repo size](https://img.shields.io/github/repo-size/StarryLab/tsid.go) ![GitHub repo file count](https://img.shields.io/github/directory-file-count/StarryLab/tsid.go) [![Go](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml/badge.svg)](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml)

English | [中文](./README.zh.md)

A unique ID generator based on a timestamp or time series, inspired by Twitter's Snowflake.

> **Woohoo!** ❗️ Timestamp segment and sequence segment is REQUIRED!

## HOWTO 🛠️

- [github.com/StarryLab/tsid.go](https://github.com/StarryLab/tsid.go)
- documentation [pkg.go.dev](https://pkg.go.dev/github.com/StarryLab/tsid.go@v1.0.0-alpha)

```bash
go get github.com/StarryLab/tsid.go
```

## FEATURES ✨

1. The maximum 126 bits
2. Customize the width of each bits segment
3. Customize the sequence of bits segments
4. Support customize encoders
5. BASE36 is default, using the go package `strconv.FormatInt`
6. An improved BASE64 encoder to encode/decode identifiers
7. Customize the options or use the provided default settings
8. Supports random or auto-increment identifiers. Note: auto-increment identifiers are still random and not strictly increment
9. Provides a classic snowflake algorithm (fixed width and position), with better performance
10. Data source types
    - Timestamps of various precision: nanosecond, millisecond, microsecond, and second
    - Various date and time values: year, month, day, week, hour, minute, second, millisecond, and the number of days and weeks in a year
    - 1 to 63 bits secure random number
    - Option value
    - Environment variables
    - Fixed value
    - Simple sequence/serial number
    - Data sources
    - Parameter by caller

## USAGE 🚀

### Example 1

```go
package main

import (
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

func main() {
  // $> ./tsid -host=8 -node=6
  host := flag.Int("host", "data center(host) id")
  node := flag.Int("node", "server node id")
  b, e := Snowflake(host, node)
  if e != nil {
    fmt.Println("Error: ", e)
    return
  }
  fmt.Println("TSID: ", b.NextString()) // strconv.FormatInt
}
```

### Example 2

```go
package main

import (
  "flag"
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

func main() {
  // $> ./tsid -host=8
  host := flag.Int("host", "data center(host) id")
  c, e := Simple(host)
  if e != nil {
    fmt.Println("Error: ", e)
    return
  }
  for i := 0; i < 100; i++ {
    fmt.Printf("%3d. %d", i+1, c())
  }
}

```

### Example 3

```go
package main

import (
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

func main() {
  // Environment variable: SERVER_HOST, SERVER_NODE
  opt := O(
    Sequence(SequenceWidth), // 12 bits, REQUIRED!
    Env(6, "SERVER_HOST", 0) // data center id, 6 bits [0, 31]
    Env(4, "SERVER_NODE", 0) // data center id, 4 bits [0, 15]
    Timestamp(TimestampWidth, TimestampMilliseconds), // 41 bits, REQUIRED!
  )
  b, e := Make(opt)
  if e != nil {
    fmt.Println("Error: ", e)
    return
  }
  fmt.Println("TSID: ", b.NextString()) // strconv.FormatInt
}
```
