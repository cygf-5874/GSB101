// Package rollhash 是文本上的重复子串索引库。
//
// 对外 API 与语义逐条写在 README 的「对外契约」一节里。
// 实现使用滚动哈希筛选候选，哈希命中后逐字节复核。
package rollhash

// Idx 是文本上建好的一版索引。内部表示完全自由，可以新增字段与方法。
type Idx struct {
	text string
	gram int
}

// Index 在 text 上建索引，k 是索引使用的窗口大小。
//
// k <= 0 时返回空索引：其 Lookup 与 DupPairs 都返回空结果。
func Index(text string, k int) *Idx {
	if k <= 0 {
		return &Idx{}
	}
	return &Idx{text: text, gram: k}
}

// Lookup 返回 pat 在文本里出现的全部起始下标：升序、去重。
//
// pat 为空、pat 比文本长、或索引为空时返回空切片。
func (ix *Idx) Lookup(pat string) []int {
	out := []int{}
	if ix.gram <= 0 {
		return out
	}
	t := ix.text
	m := len(pat)
	if m == 0 || m > len(t) {
		return out
	}

	leading := hashLeading(m)
	patHash := hashWindow(pat, 0, m)
	windowHash := hashWindow(t, 0, m)
	last := len(t) - m

	for start := 0; start <= last; start++ {
		if windowHash == patHash && sameWindow(t, start, pat, 0, m) {
			out = append(out, start)
		}
		if start < last {
			windowHash = slideWindow(windowHash, t[start], t[start+m], leading)
			stats.HashOps++
		}
	}
	return out
}

// DupPairs 返回所有相隔 k、长度为 k 的重叠重复对 [i, i+k]：按下标升序。
//
// k <= 0、索引为空、或文本短于 2k 时返回空切片。
func (ix *Idx) DupPairs(k int) [][2]int {
	out := [][2]int{}
	if k <= 0 || ix.gram <= 0 {
		return out
	}
	t := ix.text
	n := len(t)
	if n-k < k {
		return out
	}

	windowCount := n - k + 1
	hashes := make([]uint64, windowCount)
	leading := hashLeading(k)
	hashes[0] = hashWindow(t, 0, k)

	for start := 1; start < windowCount; start++ {
		hashes[start] = slideWindow(hashes[start-1], t[start-1], t[start+k-1], leading)
		stats.HashOps++
	}

	for start := 0; start+2*k <= n; start++ {
		if hashes[start] == hashes[start+k] && sameWindow(t, start, t, start+k, k) {
			out = append(out, [2]int{start, start + k})
		}
	}
	return out
}

func hashWindow(s string, start, length int) uint64 {
	var h uint64
	for i := 0; i < length; i++ {
		h = (h*hashBase + uint64(s[start+i])) & hashMask
	}
	stats.HashOps++
	return h
}

func slideWindow(h uint64, oldByte, newByte byte, leading uint64) uint64 {
	return ((h-uint64(oldByte)*leading)*hashBase + uint64(newByte)) & hashMask
}

func hashLeading(length int) uint64 {
	leading := uint64(1)
	for i := 0; i < length-1; i++ {
		leading = (leading * hashBase) & hashMask
	}
	return leading
}

func sameWindow(a string, aStart int, b string, bStart, length int) bool {
	for i := 0; i < length; i++ {
		stats.Comparisons++
		if a[aStart+i] != b[bStart+i] {
			return false
		}
	}
	return true
}
