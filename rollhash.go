// Package rollhash 是文本上的重复子串索引库。
//
// 对外 API 与语义逐条写在 README 的「对外契约」一节里。本文件是既有实现：
// 功能正确，但每次查找都在做逐字节回溯。
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
	t := ix.text
	if pat == "" || len(pat) > len(t) {
		return out
	}
	for i := 0; i+len(pat) <= len(t); i++ {
		j := 0
		for ; j < len(pat); j++ {
			stats.Comparisons++
			if t[i+j] != pat[j] {
				break
			}
		}
		if j == len(pat) {
			out = append(out, i)
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
	for i := 0; i+2*k <= len(t); i++ {
		j := 0
		for ; j < k; j++ {
			stats.Comparisons++
			if t[i+j] != t[i+k+j] {
				break
			}
		}
		if j == k {
			out = append(out, [2]int{i, i + k})
		}
	}
	return out
}
