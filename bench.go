package rollhash

// bench.go 是本题的固定入口：它给出两个工作量计数器与本题统一的窗口哈希。
// 计数器的字段名与含义是契约的一部分，不要改名、不要改含义。

// Counts 记录一段代码期间的工作量。
type Counts struct {
	// Comparisons 是「两字节是否相等」的比较次数。
	Comparisons int64
	// HashOps 是「哈希窗口推进 / 计算」的次数（算一个窗口的哈希 = 1 次）。
	HashOps int64
}

// stats 是当前累计计数；库实现里对这两个字段自增。
var stats Counts

// Stats 返回当前累计计数。
func Stats() Counts { return stats }

// Reset 把累计计数清零。
func Reset() { stats = Counts{} }

// Hash 是本题统一的窗口哈希：多项式滚动哈希，基 257，结果对 2^32 取模。
//
//	h = 0
//	for c in window:
//	    h = (h*257 + c) mod 2^32
//
// 索引里「滚动推进」出来的窗口哈希，必须与直接调用 Hash(该窗口) 的结果相同。
func Hash(b []byte) uint64 {
	var h uint64
	for _, c := range b {
		h = (h*hashBase + uint64(c)) & hashMask
	}
	return h
}

const (
	hashBase = 257
	hashMask = uint64(0xffffffff)
)
