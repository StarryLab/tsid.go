
# TSID

![GitHub](https://img.shields.io/github/license/StarryLab/tsid.go) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/StarryLab/tsid.go) [![Go Reference](https://pkg.go.dev/badge/github.com/StarryLab/tsid.go@v1.0.0-alpha.svg)](https://pkg.go.dev/github.com/StarryLab/tsid.go@v1.0.0-alpha) [![Go](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml/badge.svg)](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/StarryLab/tsid.go)](https://goreportcard.com/report/github.com/StarryLab/tsid.go) [![codecov](https://codecov.io/gh/StarryLab/tsid.go/branch/main/graph/badge.svg?token=8ENA31T0I1)](https://codecov.io/gh/StarryLab/tsid.go)
![GitHub release (latest SemVer including pre-releases)](https://img.shields.io/github/v/release/StarryLab/tsid.go?include_prereleases&sort=semver) ![GitHub last commit](https://img.shields.io/github/last-commit/StarryLab/tsid.go) ![GitHub repo size](https://img.shields.io/github/repo-size/StarryLab/tsid.go) ![GitHub repo file count](https://img.shields.io/github/directory-file-count/StarryLab/tsid.go) ![GitHub Repo stars](https://img.shields.io/github/stars/StarryLab/tsid.go?style=social)

English | [中文](./README.zh.md)

A unique ID generator based on a timestamp or time series, inspired by Twitter's Snowflake.

The goal is to provide a unique identification (or UUID) solution that is reliable and flexible (configurable, extensible), but the performance is lower than the classical (fixed width and position) snowflake algorithm. Also provides the implementation of the classic snowflake algorithm(`func Simple(server int64) func()int64`). See Example 2

> **NOTES!** ❗️Timestamp segment and sequence segment is REQUIRED!

## HOWTO 🛠️

- source code: [github.com/StarryLab/tsid.go](https://github.com/StarryLab/tsid.go)
- documentation: [pkg.go.dev](https://pkg.go.dev/github.com/StarryLab/tsid.go)

```bash
go get github.com/StarryLab/tsid.go
```

## FEATURES ✨

1. The maximum 126 bits
2. Customize the width of each bit-segment
3. Customize the order of bit-segments
4. Support customize encoder
5. BASE36 is default, using the go package `strconv.FormatInt`
6. An improved BASE64 encoder to encode/decode identifiers
7. Customize the options or use the provided default settings
8. Supports random or auto-increment identifiers. Notes: auto-increment identifiers are still random and not strictly increment
9. Provides a classic snowflake algorithm (fixed width and position), with better performance
10. Data source types
    - Timestamps of various precision: nanosecond, millisecond, microsecond, and second
    - Various date and time values: year, month, day, week, hour, minute, second, millisecond, and the number of days and weeks in a year
    - 1 to 63 bits secure random number
    - Option value
    - Environment variables
    - Fixed value
    - Simple sequence/serial number, like a counter
    - Data sources
    - Parameter by caller

## USAGE 🚀

### Example 1

```go
package main

import (
  "flag"
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

var (
  host,
  node int64
)

func init() {
  // $> ./tsid -host=8 -node=6
  host = *flag.Int64("host", 0, "data center(host) id")
  node = *flag.Int64("node", 0, "server node id")
}

func main() {
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

var (
  server int64
)

func init() {
  // $> ./tsid -server=8
  server = *flag.Int64("server", 0, "server id")
}

func main() {
  c, e := Simple(server)
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

1. examples/demo.go

```go
package examples

import (
  "errors"
  
  . "github.com/StarryLab/tsid.go"
)

func init() {
  Register("my_data_source", DemoDataSource{{
    "demo": 1,
    "other": 9,
  }})
}

type DemoDataSource struct{
  data map[string]int64
}

func(d *DemoDataSource)Read(query ...interface{}) (int64, error) {
  if len(query)>0 {
    if s, o := query[0].(string); o {
      if v, o := d.data[s]; o {
        return v, nil
      }
    }
  }
  return 0, errors.New("data not found")
}

```

2. main.go

```go
package main

import (
  "fmt"

  _ "examples"
  . "github.com/StarryLab/tsid.go"
)

func main() {
  // Environment variable: SERVER_HOST, SERVER_NODE
  opt := *O(
    Sequence(12),                         // 12 bits, REQUIRED!
    Env(6, "SERVER_HOST", 0),             // 6 bits [0, 31], data center id
    Env(4, "SERVER_NODE", 0),             // 4 bits [0, 15], server node id
    Data(10, "my_data_source", 2, "demo"),// 10 bits [0, 1023], data source
    Random(30),                           // 30 bits
    Timestamp(41, TimestampMilliseconds), // 41 bits, REQUIRED!
  )
  b, e := Make(opt)
  if e != nil {
    fmt.Println("Error: ", e)
    return
  }
  fmt.Println("TSID: ", b.NextString()) // strconv.FormatInt
}
```
