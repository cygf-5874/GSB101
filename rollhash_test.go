package rollhash

import (
	"reflect"
	"testing"
)

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

func TestLookupSingleOccurrence(t *testing.T) {
	ix := Index("hello world", 4)
	if got := ix.Lookup("world"); !eqInts(got, []int{6}) {
		t.Fatalf("Lookup(world)=%v, want [6]", got)
	}
}

func TestLookupMultipleOccurrencesAscending(t *testing.T) {
	ix := Index("ababab", 2)
	if got := ix.Lookup("ab"); !eqInts(got, []int{0, 2, 4}) {
		t.Fatalf("Lookup(ab)=%v, want [0 2 4]", got)
	}
}

func TestLookupNoOccurrence(t *testing.T) {
	ix := Index("abcdef", 3)
	if got := ix.Lookup("zzz"); len(got) != 0 {
		t.Fatalf("Lookup(zzz)=%v, want empty", got)
	}
}

func TestLookupEmptyPattern(t *testing.T) {
	ix := Index("abcdef", 3)
	if got := ix.Lookup(""); len(got) != 0 {
		t.Fatalf("Lookup(\"\")=%v, want empty", got)
	}
}

func TestLookupPatternLongerThanText(t *testing.T) {
	ix := Index("abc", 2)
	if got := ix.Lookup("abcdef"); len(got) != 0 {
		t.Fatalf("Lookup(abcdef)=%v, want empty", got)
	}
}

func TestLookupWholeText(t *testing.T) {
	ix := Index("abcabc", 3)
	if got := ix.Lookup("abcabc"); !eqInts(got, []int{0}) {
		t.Fatalf("Lookup(abcabc)=%v, want [0]", got)
	}
}

func TestLookupOverlappingOccurrences(t *testing.T) {
	ix := Index("aaaaa", 2)
	if got := ix.Lookup("aaa"); !eqInts(got, []int{0, 1, 2}) {
		t.Fatalf("Lookup(aaa)=%v, want [0 1 2]", got)
	}
}

func TestLookupRepeatedCallsStable(t *testing.T) {
	ix := Index("abcabcabc", 3)
	first := ix.Lookup("abc")
	for i := 0; i < 5; i++ {
		if got := ix.Lookup("abc"); !eqInts(got, first) {
			t.Fatalf("第 %d 次 Lookup=%v, want %v", i+2, got, first)
		}
	}
}

func TestDupPairsPeriodic(t *testing.T) {
	ix := Index("abcabcabc", 3)
	want := [][2]int{{0, 3}, {1, 4}, {2, 5}, {3, 6}}
	if got := ix.DupPairs(3); !reflect.DeepEqual(got, want) {
		t.Fatalf("DupPairs(3)=%v, want %v", got, want)
	}
}

func TestDupPairsNone(t *testing.T) {
	ix := Index("abcdefgh", 4)
	if got := ix.DupPairs(4); len(got) != 0 {
		t.Fatalf("DupPairs(4)=%v, want empty", got)
	}
}

func TestDupPairsNonPositiveK(t *testing.T) {
	ix := Index("abcabc", 3)
	if got := ix.DupPairs(0); len(got) != 0 {
		t.Fatalf("DupPairs(0)=%v, want empty", got)
	}
	if got := ix.DupPairs(-2); len(got) != 0 {
		t.Fatalf("DupPairs(-2)=%v, want empty", got)
	}
}

func TestIndexNonPositiveK(t *testing.T) {
	if got := Index("abcabc", 0).Lookup("abc"); len(got) != 0 {
		t.Fatalf("空索引 Lookup=%v, want empty", got)
	}
	if got := Index("abcabc", -1).DupPairs(3); len(got) != 0 {
		t.Fatalf("空索引 DupPairs=%v, want empty", got)
	}
}
