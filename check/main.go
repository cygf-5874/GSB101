// Command check 是 rollhash 的固定验收程序。
//
// ⚠️ 不要修改本文件。它是判定「题目有没有做对」的依据；
// 修改它只会让判定失效，不会让实现变对。
//
// 8 个场景分三组：
//
//	correctness 3：随机文本对拍 / 重复长串对拍 / 边界输入不 panic
//	cost        3：65536 规模的 Comparisons 上界 / HashOps 上界 / 计数与调用次数无关
//	collision   2：构造已知碰撞的输入下结果仍与朴素一致 / 逐字节复核被执行
//
// 判据是计数与对拍，不读墙钟、不依赖机器速度：固定件里所有随机输入都用
// 固定种子的 LCG 生成。
//
// 用法：
//
//	go run ./check                     # 跑全部场景
//	go run ./check --only cost         # 只跑一组
//	go run ./check -list               # 列出场景
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"rollhash"
)

type scenario struct {
	Group string
	Name  string
	Run   func() error
}

var scenarios = []scenario{
	{"correctness", "CR1_random_vs_naive", cr1},
	{"correctness", "CR2_repeated_long_vs_naive", cr2},
	{"correctness", "CR3_edges_no_panic", cr3},
	{"cost", "CO1_comparisons_bound", co1},
	{"cost", "CO2_hashops_bound", co2},
	{"cost", "CO3_calls_independent", co3},
	{"collision", "CL1_known_collision_matches_naive", cl1},
	{"collision", "CL2_byte_recheck_performed", cl2},
}

var groups = []string{"correctness", "cost", "collision"}

func main() {
	only := flag.String("only", "", "只跑指定组：correctness / cost / collision")
	list := flag.Bool("list", false, "列出全部场景")
	flag.Parse()

	if *list {
		for _, sc := range scenarios {
			fmt.Printf("%-12s %s\n", sc.Group, sc.Name)
		}
		return
	}

	selected := groups
	if *only != "" {
		found := false
		for _, g := range groups {
			if g == *only {
				found = true
			}
		}
		if !found {
			fmt.Printf("未知分组 %q（可选：correctness / cost / collision）\n", *only)
			os.Exit(2)
		}
		selected = []string{*only}
	}

	total, passed := 0, 0
	for _, g := range selected {
		fmt.Printf("== 组 %s ==\n", g)
		for _, sc := range scenarios {
			if sc.Group != g {
				continue
			}
			total++
			if err := runScenario(sc); err != nil {
				fmt.Printf("FAIL %s/%s  %s\n", sc.Group, sc.Name, err.Error())
				continue
			}
			passed++
			fmt.Printf("PASS %s/%s\n", sc.Group, sc.Name)
		}
	}

	fmt.Println()
	fmt.Printf("结果：通过 %d/%d\n", passed, total)
	if passed != total {
		os.Exit(1)
	}
}

func runScenario(sc scenario) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = expect("场景正常结束（不 panic）", fmt.Sprintf("panic: %v", r))
		}
	}()
	return sc.Run()
}

// ---------------------------------------------------------------- 失败描述

type detailError struct{ msg string }

func (e *detailError) Error() string { return e.msg }

func expect(want, got string) error {
	return &detailError{msg: "期望=" + want + " 实际=" + got}
}

// ------------------------------------------------------- 固定件自己的朴素实现

// naiveLookup 是与被测实现完全独立的朴素搜索，用来对拍。
func naiveLookup(text, pat string) []int {
	out := []int{}
	if pat == "" || len(pat) > len(text) {
		return out
	}
	for i := 0; i+len(pat) <= len(text); i++ {
		if text[i:i+len(pat)] == pat {
			out = append(out, i)
		}
	}
	return out
}

func naiveDupPairs(text string, k int) [][2]int {
	out := [][2]int{}
	if k <= 0 {
		return out
	}
	for i := 0; i+2*k <= len(text); i++ {
		if text[i:i+k] == text[i+k:i+2*k] {
			out = append(out, [2]int{i, i + k})
		}
	}
	return out
}

func fmtInts(a []int) string {
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func fmtPairs(a [][2]int) string {
	parts := make([]string, len(a))
	for i, p := range a {
		parts[i] = fmt.Sprintf("(%d,%d)", p[0], p[1])
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func eqInts(a, b []int) bool {
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

func eqPairs(a, b [][2]int) bool {
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

// lcg 是一个自己写的线性同余发生器：固定种子下跨运行、跨机器逐位可复现。
type lcg struct{ s uint64 }

func newLCG(seed uint64) *lcg { return &lcg{s: seed} }

func (r *lcg) next() uint64 {
	r.s = r.s*6364136223846793005 + 1442695040888963407
	return r.s
}

func (r *lcg) text(n, alphabet int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.next()%uint64(alphabet))
	}
	return string(b)
}

// checkLookup 把一次 Lookup 与朴素搜索对拍。
func checkLookup(ix *rollhash.Idx, text, pat string) error {
	got := ix.Lookup(pat)
	want := naiveLookup(text, pat)
	if !eqInts(got, want) {
		return expect(fmt.Sprintf("Lookup(%q) 与朴素搜索一致 =%s", trunc(pat), fmtInts(want)),
			fmtInts(got))
	}
	return nil
}

func checkDupPairs(ix *rollhash.Idx, text string, k int) error {
	got := ix.DupPairs(k)
	want := naiveDupPairs(text, k)
	if !eqPairs(got, want) {
		return expect(fmt.Sprintf("DupPairs(%d) 与朴素搜索一致 =%s", k, fmtPairs(want)),
			fmtPairs(got))
	}
	return nil
}

func trunc(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}
	return s
}

// ------------------------------------------------------------- correctness

func cr1() error {
	rng := newLCG(0xC0FFEE)
	for _, cfg := range []struct {
		n, alphabet, patLen int
	}{
		{512, 2, 3}, {1024, 3, 5}, {2048, 4, 7}, {4096, 8, 11}, {4096, 26, 17},
	} {
		text := rng.text(cfg.n, cfg.alphabet)
		ix := rollhash.Index(text, 4)
		// 真子串：一定出现。
		for off := 0; off+cfg.patLen <= cfg.n; off += cfg.n / 3 {
			if err := checkLookup(ix, text, text[off:off+cfg.patLen]); err != nil {
				return err
			}
		}
		// 随机模式：多半不出现。
		for r := 0; r < 4; r++ {
			if err := checkLookup(ix, text, rng.text(cfg.patLen, cfg.alphabet)); err != nil {
				return err
			}
		}
		for _, k := range []int{1, 2, 3, 5, 8, 13} {
			if err := checkDupPairs(ix, text, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func cr2() error {
	texts := []string{
		strings.Repeat("ab", 2048),
		strings.Repeat("a", 3000) + "b" + strings.Repeat("a", 1000),
		strings.Repeat("abcabcabd", 400),
		strings.Repeat("x", 4096),
		"mississippi" + strings.Repeat("ssippi", 200),
	}
	for _, text := range texts {
		ix := rollhash.Index(text, 8)
		for _, pat := range []string{
			"ab", "aab", "abcabcabd", "ssippi", "a", "x", "abcd", text[:min(len(text), 32)],
		} {
			if err := checkLookup(ix, text, pat); err != nil {
				return err
			}
		}
		for _, k := range []int{1, 2, 3, 4, 9} {
			if err := checkDupPairs(ix, text, k); err != nil {
				return err
			}
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cr3() error {
	// 空文本。
	empty := rollhash.Index("", 4)
	if err := checkLookup(empty, "", "a"); err != nil {
		return err
	}
	if got := empty.DupPairs(2); len(got) != 0 {
		return expect("空文本 DupPairs 为空", fmtPairs(got))
	}
	// 空模式。
	ix := rollhash.Index("abcdef", 4)
	if got := ix.Lookup(""); len(got) != 0 {
		return expect("空模式的 Lookup 为空", fmtInts(got))
	}
	// 模式比文本长。
	if got := ix.Lookup("abcdefgh"); len(got) != 0 {
		return expect("模式长于文本时 Lookup 为空", fmtInts(got))
	}
	// 空索引（k <= 0）。
	k0 := rollhash.Index("abcabcabc", 0)
	if got := k0.Lookup("abc"); len(got) != 0 {
		return expect("空索引的 Lookup 为空", fmtInts(got))
	}
	if got := k0.DupPairs(3); len(got) != 0 {
		return expect("空索引的 DupPairs 为空", fmtPairs(got))
	}
	// DupPairs 的 k <= 0。
	if got := ix.DupPairs(0); len(got) != 0 {
		return expect("DupPairs(0) 为空", fmtPairs(got))
	}
	// 全同字符、模式恰为整段文本。
	same := rollhash.Index(strings.Repeat("z", 300), 4)
	if err := checkLookup(same, strings.Repeat("z", 300), strings.Repeat("z", 300)); err != nil {
		return err
	}
	if err := checkDupPairs(same, strings.Repeat("z", 300), 7); err != nil {
		return err
	}
	return nil
}

// --------------------------------------------------------------------- cost

const costN = 65536

// costText/costPat 是本次代价场景的固定输入：全 'a' 文本 + 只差最后一字节的模式。
// 朴素实现要在每个位置都比满 63 个字节才发现不匹配。
var (
	costText = strings.Repeat("a", costN)
	costPat  = strings.Repeat("a", 63) + "b"
)

const comparisonsBound = 200000

func costRun() rollhash.Counts {
	rollhash.Reset()
	ix := rollhash.Index(costText, 8)
	got := ix.Lookup(costPat)
	if len(got) != 0 {
		panic(fmt.Sprintf("代价场景的模式不该命中：%s", fmtInts(got)))
	}
	return rollhash.Stats()
}

func co1() error {
	c := costRun()
	if c.Comparisons > comparisonsBound {
		return expect(fmt.Sprintf("len(text)=%d 时 Comparisons<=%d", costN, comparisonsBound),
			fmt.Sprintf("%d", c.Comparisons))
	}
	return nil
}

func co2() error {
	c := costRun()
	bound := int64(4 * costN)
	if c.HashOps <= 0 {
		return expect("HashOps>0（查找要用到滚动哈希）", fmt.Sprintf("%d", c.HashOps))
	}
	if c.HashOps > bound {
		return expect(fmt.Sprintf("len(text)=%d 时 HashOps<=%d", costN, bound),
			fmt.Sprintf("%d", c.HashOps))
	}
	return nil
}

func co3() error {
	ix := rollhash.Index(costText, 8)
	first := ix.Lookup(costPat)

	var delta rollhash.Counts
	for i := 0; i < 5; i++ {
		rollhash.Reset()
		got := ix.Lookup(costPat)
		if !eqInts(got, first) {
			return expect(fmt.Sprintf("第 %d 次 Lookup 与第 1 次结果相同 %s", i+2, fmtInts(first)),
				fmtInts(got))
		}
		// Reset 之后读到的就是这一次 Lookup 的计数增量。
		d := rollhash.Stats()
		if i == 0 {
			delta = d
			if delta.Comparisons > comparisonsBound {
				return expect(fmt.Sprintf("单次 Lookup 的 Comparisons<=%d", comparisonsBound),
					fmt.Sprintf("%d", delta.Comparisons))
			}
		} else if d != delta {
			return expect(fmt.Sprintf("每次 Lookup 的计数相同 Comparisons=%d HashOps=%d",
				delta.Comparisons, delta.HashOps),
				fmt.Sprintf("Comparisons=%d HashOps=%d", d.Comparisons, d.HashOps))
		}
	}
	return nil
}

// ---------------------------------------------------------------- collision

// colliding 返回两个不同的、长度相同的窗口，它们的 Hash 相同。
// 用固定种子的 LCG 生成，跨运行可复现；不依赖被测实现内部结构，只用 bench.go 的 Hash。
func colliding() (a, b []byte) {
	const n = 8
	seen := make(map[uint64][]byte)
	rng := newLCG(0x1234_5678_9ABC_DEF0)
	buf := make([]byte, n)
	for iter := 0; iter < 1<<24; iter++ {
		for i := 0; i < n; i++ {
			buf[i] = byte('A' + rng.next()%26)
		}
		w := append([]byte(nil), buf...)
		h := rollhash.Hash(w)
		if prev, ok := seen[h]; ok && !bytes.Equal(prev, w) {
			return prev, w
		}
		seen[h] = w
	}
	panic("没有在预算内找到碰撞（不该发生）")
}

var (
	colA, colB = "", ""
)

func collisionStrings() (string, string) {
	if colA == "" {
		a, b := colliding()
		colA, colB = string(a), string(b)
	}
	return colA, colB
}

func cl1() error {
	w1, w2 := collisionStrings()
	if rollhash.Hash([]byte(w1)) != rollhash.Hash([]byte(w2)) {
		return expect("两个对照窗口的 Hash 相同", "Hash 不同")
	}
	text := w1 + w2
	rollhash.Reset()
	ix := rollhash.Index(text, len(w1))
	if err := checkLookup(ix, text, w1); err != nil {
		return err
	}
	if err := checkLookup(ix, text, w2); err != nil {
		return err
	}
	if rollhash.Stats().HashOps <= 0 {
		return expect("碰撞输入下的 HashOps>0（查找用到了滚动哈希）",
			fmt.Sprintf("%d", rollhash.Stats().HashOps))
	}
	return nil
}

func cl2() error {
	w1, w2 := collisionStrings()
	text := w1 + w2
	rollhash.Reset()
	ix := rollhash.Index(text, len(w1))

	got := ix.Lookup(w1)
	want := naiveLookup(text, w1)
	if !eqInts(got, want) {
		return expect(fmt.Sprintf("碰撞输入下 Lookup(%q) 与朴素一致 =%s", w1, fmtInts(want)),
			fmtInts(got))
	}
	// 哈希相同但字节不同的那个位置绝不能报出来。
	for _, p := range got {
		if p == len(w1) {
			return expect("哈希相同但字节不同的位置被逐字节复核挡住",
				fmt.Sprintf("报出了位置 %d", p))
		}
	}
	st := rollhash.Stats()
	if st.HashOps <= 0 {
		return expect("HashOps>0", fmt.Sprintf("%d", st.HashOps))
	}
	if st.Comparisons <= 0 {
		return expect("命中哈希后逐字节复核（Comparisons>0）", fmt.Sprintf("%d", st.Comparisons))
	}
	return nil
}
