package bm25

import (
	"fmt"
	"testing"
)

func TestCorpus_RanksTermDenseFirst(t *testing.T) {
	c := New()
	c.Add("dense", "transformer attention transformer transformer architecture")
	c.Add("mention", "a passing mention of one transformer somewhere in here")
	c.Add("unrelated", "recipes ingredients oven baking flour sugar")

	got := c.Search("transformer architecture", 0)
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3", len(got))
	}
	if got[0].ID != "dense" {
		t.Errorf("top = %q, want dense; results=%+v", got[0].ID, got)
	}
	if got[0].Score <= got[1].Score {
		t.Errorf("scores not descending: %+v", got)
	}
	if last := got[len(got)-1]; last.ID != "unrelated" || last.Score != 0 {
		t.Errorf("last = %+v, want unrelated with score 0", last)
	}
}

func TestCorpus_TopK(t *testing.T) {
	c := New()
	for i := 0; i < 5; i++ {
		c.Add(fmt.Sprintf("d%d", i), "alpha beta gamma")
	}
	if got := c.Search("alpha", 2); len(got) != 2 {
		t.Errorf("top-2 returned %d results", len(got))
	}
	if got := c.Search("alpha", 0); len(got) != 5 {
		t.Errorf("k<=0 returned %d results, want all 5", len(got))
	}
}

func TestCorpus_StableTieOrder(t *testing.T) {
	c := New()
	// Identical bodies → identical scores → must order by ID.
	c.Add("b", "same words here")
	c.Add("a", "same words here")
	got := c.Search("same words", 0)
	if got[0].ID != "a" || got[1].ID != "b" {
		t.Errorf("tie order = %q,%q; want a,b (stable by ID)", got[0].ID, got[1].ID)
	}
}

func TestFuseRanked(t *testing.T) {
	// a is #1 in both, d last in both; b/c swap the middle.
	got := FuseRanked(DefaultRRFK,
		[]string{"a", "b", "c", "d"},
		[]string{"a", "c", "b", "d"},
	)
	if got[0].ID != "a" {
		t.Errorf("FuseRanked top = %q, want a (best in both)", got[0].ID)
	}
	if got[len(got)-1].ID != "d" {
		t.Errorf("FuseRanked last = %q, want d (worst in both)", got[len(got)-1].ID)
	}
	// Sorted descending.
	for i := 1; i < len(got); i++ {
		if got[i].Score > got[i-1].Score {
			t.Errorf("FuseRanked not sorted desc: %+v", got)
		}
	}
}

// Example_bm25 shows ranking a small corpus by keyword relevance.
func Example_bm25() {
	c := New()
	c.Add("moby", "Call me Ishmael. The whale, the ship, the sea.")
	c.Add("austen", "It is a truth universally acknowledged, marriage and fortune.")

	for _, r := range c.Search("whale ship", 0) {
		if r.Score > 0 {
			fmt.Println(r.ID)
		}
	}
	// Output: moby
}

// Example_hybridRRF shows fusing a keyword ranking with a (pretend)
// vector-similarity ranking via reciprocal-rank fusion.
func Example_hybridRRF() {
	keyword := []string{"doc1", "doc2", "doc3"}  // best-first by BM25
	semantic := []string{"doc1", "doc3", "doc4"} // best-first by cosine

	fused := FuseRanked(DefaultRRFK, keyword, semantic)
	fmt.Println(fused[0].ID) // doc1 is #1 in both → ranks first
	// Output: doc1
}
