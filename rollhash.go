// Package rollhash 是文本上的重复子串索引库。
//
// 对外 API 与语义逐条写在 README 的「对外契约」一节里。本版实现不在每个
// 位置逐字节回溯：建库时滚动出全部窗口哈希，查找时先按哈希圈定候选，再对
// 候选逐字节复核（见 README 契约 7）。
package rollhash

// Idx 是文本上建好的一版索引。内部表示完全自由，可以新增字段与方法。
type Idx struct {
	tb    []byte
	gram  int
	win   []uint32         // win[i] 等于 Hash(tb[i:i+gram])，长度为 n-gram+1
	posts map[uint32][]int // 哈希值 -> 该值出现的所有窗口起点，按位置升序
}

// Index 在 text 上建索引，k 是索引使用的窗口大小。
//
// k <= 0 时返回空索引：其 Lookup 与 DupPairs 都返回空结果。
func Index(text string, k int) *Idx {
	if k <= 0 {
		return &Idx{}
	}
	t := []byte(text)
	n := len(t)
	ix := &Idx{tb: t, gram: k}
	if n >= k {
		w := n - k + 1
		ix.win = make([]uint32, w)
		ix.posts = make(map[uint32][]int, w)

		var h uint32
		for i := 0; i < k; i++ {
			h = h*hashBase + uint32(t[i])
		}
		stats.HashOps++
		ix.win[0] = h

		basePow := powBase(k - 1)
		for i := 1; i < w; i++ {
			h = (h-uint32(t[i-1])*basePow)*hashBase + uint32(t[i+k-1])
			stats.HashOps++
			ix.win[i] = h
		}
		for i, hv := range ix.win {
			ix.posts[hv] = append(ix.posts[hv], i)
		}
	}
	return ix
}

// powBase 返回 hashBase^p 对 2^32 取模的结果。
func powBase(p int) uint32 {
	var r uint64 = 1
	var b uint64 = hashBase
	for p > 0 {
		if p&1 == 1 {
			r = r * b & hashMask
		}
		b = b * b & hashMask
		p >>= 1
	}
	return uint32(r)
}

// Lookup 返回 pat 在文本里出现的全部起始下标：升序、去重。
//
// pat 为空、pat 比文本长、或索引为空时返回空切片。
func (ix *Idx) Lookup(pat string) []int {
	out := []int{}
	if ix.gram <= 0 {
		return out
	}
	t := ix.tb
	n, m := len(t), len(pat)
	if m == 0 || m > n {
		return out
	}
	k := ix.gram
	if m < k {
		// 模式比索引窗口窄：按模式宽度做一遍滚动哈希，命中后逐字节复核。
		pb := []byte(pat)
		ph := uint32(Hash(pb))
		stats.HashOps++
		basePow := powBase(m - 1)
		var h uint32
		for i := 0; i < m; i++ {
			h = h*hashBase + uint32(t[i])
		}
		stats.HashOps++
		for i := 0; i+m <= n; i++ {
			if h == ph && bytesEqualCounted(t[i:i+m], pb) {
				out = append(out, i)
			}
			if i+m < n {
				h = (h-uint32(t[i])*basePow)*hashBase + uint32(t[i+m])
				stats.HashOps++
			}
		}
		return out
	}

	// 模式不窄于索引窗口：用模式内部最稀有的 k-gram 作探针。
	// 它在索引桶中命中的位置平移回模式起点，就是全部候选；
	// 桶本身按位置升序，所以候选天然升序、去重。
	pb := []byte(pat)
	bestOff := 0
	h := uint32(Hash(pb[:k]))
	stats.HashOps++
	bestList := ix.posts[h]
	basePow := powBase(k - 1)
	for off := 1; off+k <= m; off++ {
		h = (h-uint32(pb[off-1])*basePow)*hashBase + uint32(pb[off+k-1])
		stats.HashOps++
		list := ix.posts[h]
		if len(list) < len(bestList) {
			bestOff = off
			bestList = list
		}
	}
	for _, p := range bestList {
		start := p - bestOff
		if start < 0 || start+m > n {
			continue
		}
		if bytesEqualCounted(t[start:start+m], pb) {
			out = append(out, start)
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
	t := ix.tb
	n := len(t)
	if n < 2*k {
		return out
	}

	// hh[i] 是 text[i:i+k] 的哈希。k 等于建库窗口时直接复用，否则现滚一遍。
	if k == ix.gram {
		for i := 0; i+k < len(ix.win); i++ {
			if ix.win[i] != ix.win[i+k] {
				continue
			}
			if bytesEqualCounted(t[i:i+k], t[i+k:i+2*k]) {
				out = append(out, [2]int{i, i + k})
			}
		}
		return out
	}

	p := n - k + 1
	hh := make([]uint32, p)
	var h uint32
	for i := 0; i < k; i++ {
		h = h*hashBase + uint32(t[i])
	}
	stats.HashOps++
	hh[0] = h
	basePow := powBase(k - 1)
	for i := 1; i < p; i++ {
		h = (h-uint32(t[i-1])*basePow)*hashBase + uint32(t[i+k-1])
		stats.HashOps++
		hh[i] = h
	}
	for i := 0; i+2*k <= n; i++ {
		if hh[i] != hh[i+k] {
			continue
		}
		if bytesEqualCounted(t[i:i+k], t[i+k:i+2*k]) {
			out = append(out, [2]int{i, i + k})
		}
	}
	return out
}

// bytesEqualCounted 逐字节比较两个等长切片，每比一对字节记一次 Comparisons。
func bytesEqualCounted(a, b []byte) bool {
	for i := range a {
		stats.Comparisons++
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
