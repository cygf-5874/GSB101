package rollhash

import (
	"testing"
)

// TestRolledWindowHashesMatchHash 守护契约 7：索引里滚动出来的每个窗口哈希
// 必须与直接调用 bench.go 的 Hash 完全一致。
func TestRolledWindowHashesMatchHash(t *testing.T) {
	rng := newTestLCG(0x5EED)
	for _, cfg := range []struct{ n, k, alpha int }{
		{64, 1, 2}, {64, 2, 3}, {100, 4, 4}, {200, 8, 26},
		{10, 10, 2}, {9, 10, 2}, {50, 13, 5}, {4096, 8, 26},
	} {
		text := rng.text(cfg.n, cfg.alpha)
		ix := Index(text, cfg.k)
		tb := []byte(text)
		if cfg.n >= cfg.k {
			if len(ix.win) != cfg.n-cfg.k+1 {
				t.Fatalf("n=%d k=%d: win 长度=%d", cfg.n, cfg.k, len(ix.win))
			}
			for i := range ix.win {
				want := uint32(Hash(tb[i : i+cfg.k]))
				if ix.win[i] != want {
					t.Fatalf("n=%d k=%d 窗口 %d: 滚动=%08x Hash=%08x",
						cfg.n, cfg.k, i, ix.win[i], want)
				}
			}
			for hv, ps := range ix.posts {
				for _, p := range ps {
					if ix.win[p] != hv {
						t.Fatalf("桶 %08x 中位置 %d 的窗口哈希不符", hv, p)
					}
				}
			}
		} else if len(ix.win) != 0 {
			t.Fatalf("n<k 时 win 应为空")
		}
	}
}

// TestExhaustiveVsNaive 穷举短文本上的全部模式长度与多种 gram，覆盖
// m < gram 的 Rabin-Karp 兜底分支和 m >= gram 的稀有探针分支。
func TestExhaustiveVsNaive(t *testing.T) {
	rng := newTestLCG(0xABCD)
	for _, cfg := range []struct{ n, alpha int }{
		{40, 2}, {60, 3}, {80, 4},
	} {
		text := rng.text(cfg.n, cfg.alpha)
		for _, k := range []int{1, 2, 4, 8} {
			ix := Index(text, k)
			for m := 1; m <= cfg.n; m++ {
				for s := 0; s+m <= cfg.n; s++ {
					pat := text[s : s+m]
					if !eqInts(ix.Lookup(pat), naiveTestLookup(text, pat)) {
						t.Fatalf("text=%q k=%d Lookup(%q) 与朴素不一致", text, k, pat)
					}
				}
				pat := rng.text(m, cfg.alpha)
				if !eqInts(ix.Lookup(pat), naiveTestLookup(text, pat)) {
					t.Fatalf("text=%q k=%d Lookup(随机 %q) 与朴素不一致", text, k, pat)
				}
			}
		}
	}
}

// TestExhaustiveDupPairsVsNaive 穷举 DupPairs 的全部周期。
func TestExhaustiveDupPairsVsNaive(t *testing.T) {
	texts := []string{
		"",
		"a",
		"ab",
		"abcabcabc",
		"aaaaaaaaaaaaaaaa",
		"mississippimississippi",
	}
	rng := newTestLCG(7)
	for a := 0; a < 4; a++ {
		texts = append(texts, rng.text(120, a+1))
	}
	for _, text := range texts {
		for _, k := range []int{1, 2, 3, 4, 8} {
			ix := Index(text, k)
			for d := 1; d <= 12; d++ {
				got := ix.DupPairs(d)
				want := naiveTestDupPairs(text, d)
				if !eqPairsTest(got, want) {
					t.Fatalf("text=%q gram=%d DupPairs(%d)=%v want %v",
						text, k, d, got, want)
				}
			}
			if got := ix.DupPairs(0); len(got) != 0 {
				t.Fatalf("DupPairs(0) 应为空")
			}
		}
	}
}

// TestLookupCountsDeterministic 同一索引上重复 Lookup 的结果与计数完全一致。
func TestLookupCountsDeterministic(t *testing.T) {
	text := rngTextRepeat(2048)
	ix := Index(text, 8)
	Reset()
	first := ix.Lookup(text[100:164])
	base := Stats()
	for r := 0; r < 5; r++ {
		Reset()
		got := ix.Lookup(text[100:164])
		if !eqInts(got, first) {
			t.Fatalf("第 %d 次结果不同", r+2)
		}
		if Stats() != base {
			t.Fatalf("第 %d 次计数不同: %v vs %v", r+2, Stats(), base)
		}
	}
}

type testLCG struct{ s uint64 }

func newTestLCG(seed uint64) *testLCG { return &testLCG{s: seed} }

func (r *testLCG) next() uint64 {
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return r.s
}

func (r *testLCG) text(n, alphabet int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.next()%uint64(alphabet))
	}
	return string(b)
}

func naiveTestLookup(text, pat string) []int {
	var out []int
	for i := 0; i+len(pat) <= len(text); i++ {
		if text[i:i+len(pat)] == pat {
			out = append(out, i)
		}
	}
	return out
}

func naiveTestDupPairs(text string, k int) [][2]int {
	var out [][2]int
	for i := 0; i+2*k <= len(text); i++ {
		if text[i:i+k] == text[i+k:i+2*k] {
			out = append(out, [2]int{i, i + k})
		}
	}
	return out
}

func eqPairsTest(a, b [][2]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func rngTextRepeat(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = "abcde"[i%5]
	}
	return string(b)
}
