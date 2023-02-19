
# 基于时序的唯一标识符（雪花算法）

[English](./README.md) | 中文

![GitHub](https://img.shields.io/github/license/StarryLab/tsid.go) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/StarryLab/tsid.go) [![Go Reference](https://pkg.go.dev/badge/github.com/StarryLab/tsid.go@v1.0.0-alpha.svg)](https://pkg.go.dev/github.com/StarryLab/tsid.go@v1.0.0-alpha) [![Go](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml/badge.svg)](https://github.com/StarryLab/tsid.go/actions/workflows/go.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/StarryLab/tsid.go)](https://goreportcard.com/report/github.com/StarryLab/tsid.go)
![GitHub release (latest SemVer including pre-releases)](https://img.shields.io/github/v/release/StarryLab/tsid.go?include_prereleases&sort=semver) ![GitHub last commit](https://img.shields.io/github/last-commit/StarryLab/tsid.go) ![GitHub repo size](https://img.shields.io/github/repo-size/StarryLab/tsid.go) ![GitHub repo file count](https://img.shields.io/github/directory-file-count/StarryLab/tsid.go) ![GitHub Repo stars](https://img.shields.io/github/stars/StarryLab/tsid.go?style=social)

根据 Twitter 的雪花算法思想开发的唯一标识符生成器，相较于已有雪花算法作了很多改进和扩展，目的是提供一套可靠和灵活（可配置、可扩展）的唯一标识解决方案，但在性能上不如经典的（固定宽度和位置）雪花算法。同时也提供了经典雪花算法方法 `func Single(server int64) func()int64`，参考用法例 2

> **注意** ❗️ 选项中必须包括时间戳（任意精度）及序号类型的位段

## 开始 🛠️

- 源码库 [github.com/StarryLab/tsid.go](https://github.com/StarryLab/tsid.go)
- 文档 [pkg.go.dev](https://pkg.go.dev/github.com/StarryLab/tsid.go@v1.0.0-alpha)

```bash
go get github.com/StarryLab/tsid.go
```

## 特点 ✨

1. 最大有效数据位可达 126 位，即两个 uint64 位宽
2. 指定每个数据位段的宽度
3. 调整数据位段的顺序
4. 支持自定义的编码器
5. 默认使用 BASE36 编码，使用 go 包 `strconv.FormatInt`
6. 提供改进的 BASE64 编码器对标识符进行编码/解码
7. 自定义选项配置，或者直接使用已提供的默认配置
8. 支持随机或趋势递增两种形式的标识符。注意：趋势递增的标识符仍然是随机的，非严格递增
9. 提供传统雪花算法的方法（固定宽度和位置），性能较好，约是可变算法的 4~5 倍
10. 提供多种数据来源类型以满足丰富的需求
    - 各种精度的时间戳：纳秒、毫秒、微秒及秒
    - 各种日期时间值：年、月、日、周、时、分、秒、毫秒，还有一年内的天数和周数
    - 1~63 位宽的安全随机数
    - 选项值
    - 环境变量
    - 定值
    - 简单序列号
    - 外部数据源
    - 调用时传入的参数

## 用法 🚀

### 例 1: 基本用法

```go
package main

import (
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

func main() {
  // 来自命令行的参数
  // $> ./tsid -host=8 -node=6
  host := flag.Int("host", "data center(host) id")
  node := flag.Int("node", "server node id")
  b, e := Snowflake(host, node)
  if e != nil {
    fmt.Println("发生错误: ", e)
    return
  }
  // 生成标识符，使用 BASE36 编码
  fmt.Println("TSID: ", b.NextString())
}
```

### 例 2: 简单的雪花算法

```go
package main

import (
  "flag"
  "fmt"

  . "github.com/StarryLab/tsid.go"
)

func main() {
  // 来自命令行的参数
  // $> ./tsid -host=8
  host := flag.Int("host", "data center(host) id")
  c, e := Simple(host)
  if e != nil {
    fmt.Println("发生错误: ", e)
    return
  }
  for i := 0; i < 100; i++ {
    fmt.Printf("%3d. %d", i+1, c())
  }
}

```

### 例 3: 自定义位段宽度及顺序

1. `examples/demo.go`

```go
package examples

import (
  "github.com/StarryLab/tsid.go"
)

func init() {
  tsid.Register("my_data_source", DemoDataSource{map[string]int64{
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

2. `main.go`

```go
package main

import (
  "fmt"

  _ "examples"
  . "github.com/StarryLab/tsid.go"
)

func main() {
  // 环境变量 SERVER_HOST 和 SERVER_NODE 指定数据中心和服务器节点号
  opt := O(
    Sequence(12),                         // 12 bits, REQUIRED!
    Env(6, "SERVER_HOST", 0)              // 6 bits [0, 31] 数据中心编号
    Env(4, "SERVER_NODE", 0)              // 4 bits [0, 15] 服务器节点编号
    Data(10, "my_data_source", 2, "demo") // 10 bits [0, 1023]
    Random(30),                           // 30 bits
    Timestamp(41, TimestampMilliseconds), // 41 bits, REQUIRED!
  )
  b, e := Make(opt)
  if e != nil {
    fmt.Println("发生错误: ", e)
    return
  }
  // 生成标识符，使用 BASE36 编码
  fmt.Println("TSID: ", b.NextString())
}
```
