# bm25

[![Go Reference](https://pkg.go.dev/badge/github.com/richardwooding/bm25.svg)](https://pkg.go.dev/github.com/richardwooding/bm25)
[![CI](https://github.com/richardwooding/bm25/actions/workflows/ci.yml/badge.svg)](https://github.com/richardwooding/bm25/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/richardwooding/bm25)](https://goreportcard.com/report/github.com/richardwooding/bm25)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Website:** [richardwooding.github.io/bm25](https://richardwooding.github.io/bm25/)

Okapi **BM25** keyword-relevance ranking and **reciprocal-rank fusion (RRF)** for Go —
small, dependency-free, and built for the keyword half of hybrid (keyword + semantic)
search.

- **Zero third-party dependencies** — `math` / `strings` / `unicode` only.
- **`go 1.21`+** — broad compatibility, no surprises.
- **Two layers:** a one-call `Corpus` for the common case, and the low-level `Scorer`
  primitives when you need to tokenize yourself or score against a per-query candidate set.
- **RRF built in** — fuse a BM25 ranking with a vector-similarity (or any other) ranking,
  the standard recipe for hybrid search, with no weights to tune.

```sh
go get github.com/richardwooding/bm25
```

## Keyword ranking

```go
c := bm25.New()
c.Add("moby",   "Call me Ishmael. The whale, the ship, the sea.")
c.Add("austen", "It is a truth universally acknowledged, marriage and fortune.")

for _, r := range c.Search("whale ship", 5) {
    fmt.Printf("%.3f  %s\n", r.Score, r.ID)
}
// 1.39  moby
```

`Search(query, k)` returns the top `k` documents by BM25 score, descending (ties broken by
ID); `k <= 0` returns all. IDF and average document length are computed over the whole
corpus — the standard Okapi BM25 setup. Tune the parameters with
`bm25.NewWithParams(k1, b)`.

## Hybrid search (reciprocal-rank fusion)

Keyword search is precise on exact terms; vector search catches paraphrase. **RRF** blends
two ranked lists into one without picking weights:

```go
keyword  := c.Search("http caching proxies", 0)         // []bm25.Result
semantic := myVectorIndex.Search("http caching", 0)     // your embedding ranker

// Fuse the two orderings (best-first ID lists) into one ranking.
fused := bm25.FuseRanked(bm25.DefaultRRFK,
    ids(keyword),   // []string of IDs, best-first
    ids(semantic),
)
fmt.Println(fused[0].ID) // strong in both lists ranks first
```

`ReciprocalRankFusion` returns the raw `map[id]score` if you'd rather sort yourself.

## Bring your own tokenizer

`Corpus` uses a deliberately simple built-in tokenizer (lowercased Unicode
letter/digit runs). For stemming, stop-words, or n-grams, tokenize however you like and
drive the low-level API:

```go
queryTerms := myTokenize(query)
docs := make([]bm25.DocStats, len(corpus))
for i, d := range corpus {
    docs[i] = bm25.StatsForTokens(myTokenize(d.text), queryTerms)
}
scorer := bm25.NewScorer(queryTerms, docs, bm25.DefaultK1, bm25.DefaultB)
score := scorer.Score(docs[0])
```

This low-level path also lets you compute IDF over a **per-query candidate set** (e.g. only
the documents that passed an upstream filter) rather than the whole corpus.

## License

MIT — see [LICENSE](LICENSE).

---

Extracted from [file-search-on](https://github.com/richardwooding/file-search-on), where it
powers hybrid keyword + semantic file search.
