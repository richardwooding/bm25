package bm25

import "sort"

// Result is one ranked document: its caller-supplied ID and score.
// Higher Score is more relevant. Scores are only comparable within a
// single Search / FuseRanked call.
type Result struct {
	ID    string
	Score float64
}

// Corpus is a small in-memory BM25 index for the common "add documents
// once, run many queries" workflow. IDF and average document length are
// computed over the WHOLE corpus (the standard Okapi BM25 setup).
//
// For advanced use — streaming, or scoring against a per-query candidate
// set whose IDF should reflect only those candidates — drive the
// lower-level Scorer / DocStats / StatsForTokens directly instead.
//
// A Corpus is not safe for concurrent Add and Search; guard it yourself
// if you mutate and query from multiple goroutines.
type Corpus struct {
	k1, b   float64
	ids     []string
	docToks [][]string // each document tokenized once at Add time
}

// New returns an empty Corpus using the default BM25 parameters
// (DefaultK1, DefaultB).
func New() *Corpus { return &Corpus{k1: DefaultK1, b: DefaultB} }

// NewWithParams returns an empty Corpus with explicit BM25 parameters.
// k1 controls term-frequency saturation, b the length normalisation
// (0..1). Out-of-range values fall back to the defaults at score time.
func NewWithParams(k1, b float64) *Corpus { return &Corpus{k1: k1, b: b} }

// Add indexes a document under id, tokenizing its text with the built-in
// Tokenize. To use your own tokenizer (stemming, stop-words, n-grams),
// score with the lower-level Scorer instead.
func (c *Corpus) Add(id, text string) {
	c.ids = append(c.ids, id)
	c.docToks = append(c.docToks, Tokenize(text))
}

// Len is the number of indexed documents.
func (c *Corpus) Len() int { return len(c.ids) }

// Search ranks the corpus against query and returns the top k results by
// BM25 score, descending (ties broken by ID for a stable order). k <= 0
// returns every document. Documents with no query-term hits score 0.
func (c *Corpus) Search(query string, k int) []Result {
	terms := Tokenize(query)
	docs := make([]DocStats, len(c.docToks))
	for i, toks := range c.docToks {
		docs[i] = StatsForTokens(toks, terms)
	}
	scorer := NewScorer(terms, docs, c.k1, c.b)

	out := make([]Result, len(docs))
	for i := range docs {
		out[i] = Result{ID: c.ids[i], Score: scorer.Score(docs[i])}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Score != out[b].Score {
			return out[a].Score > out[b].Score
		}
		return out[a].ID < out[b].ID
	})
	if k > 0 && len(out) > k {
		out = out[:k]
	}
	return out
}

// FuseRanked combines several ranked ID lists into one, scored by
// reciprocal-rank fusion, and returns them sorted by fused score
// descending (ties broken by ID). It's the sorted-slice companion to
// ReciprocalRankFusion — the shape most callers want for hybrid search
// (e.g. fuse a BM25 ranking with a vector-similarity ranking). k is the
// RRF damping constant (DefaultRRFK is the usual choice). Each ranking
// is best-first; an ID missing from a list simply doesn't get that
// list's contribution.
func FuseRanked(k float64, rankings ...[]string) []Result {
	fused := ReciprocalRankFusion(k, rankings...)
	out := make([]Result, 0, len(fused))
	for id, score := range fused {
		out = append(out, Result{ID: id, Score: score})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Score != out[b].Score {
			return out[a].Score > out[b].Score
		}
		return out[a].ID < out[b].ID
	})
	return out
}
