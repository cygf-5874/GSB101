// Command repro 打印 rollhash 在几个基准规模下的计数。
//
// 文本全为 'a'，模式是「63 个 a 加一个 b」：朴素实现要在每个位置都比满
// 63 个字节才发现不匹配；滚动哈希的实现只推进窗口、不逐字节回溯。
//
// 跑法：
//
//	go run ./repro
package main

import (
	"fmt"
	"strings"

	"rollhash"
)

func run(n, m int) {
	text := strings.Repeat("a", n)
	pat := strings.Repeat("a", m-1) + "b"

	rollhash.Reset()
	ix := rollhash.Index(text, 8)
	got := ix.Lookup(pat)
	c := rollhash.Stats()

	fmt.Printf("len(text)=%6d len(pat)=%2d: Comparisons=%10d HashOps=%10d 命中=%d\n",
		n, m, c.Comparisons, c.HashOps, len(got))
}

func main() {
	fmt.Println("Lookup 的计数（模式=[a*(m-1)+b]，文本全为 a）：")
	for _, n := range []int{4096, 16384, 65536} {
		run(n, 64)
	}
}
