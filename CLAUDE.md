# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build ./...                                    # build
go vet ./...                                       # vet (CI gate)
go test -race ./...                                # test with race detector (matches CI)
go test -run TestBM25Scoring ./...                 # run a single test by name
golangci-lint run                                  # lint (CI uses golangci-lint, version: latest)
```

CI (`.github/workflows/ci.yml`) builds/vets/tests against Go `1.21` and `stable`, and runs `golangci-lint`. The `1.21` matrix entry exists to prove the declared `go.mod` floor still compiles — don't use language/stdlib features newer than 1.21.

## Architecture

A small, **zero-dependency** (`math`/`strings`/`unicode` only) library for Okapi BM25 keyword ranking plus reciprocal-rank fusion. It's the keyword half of hybrid (keyword + semantic) search, extracted from [file-search-on](https://github.com/richardwooding/file-search-on).

Two layers, both in package `bm25`:

- **High-level `Corpus`** (`corpus.go`): the "add once, query many" path. `New()`/`NewWithParams(k1,b)` → `Add(id, text)` → `Search(query, k)`. Documents are tokenized once at `Add` time and stored as `docToks`. IDF and avgdl are computed over the **whole corpus**. Not safe for concurrent `Add`/`Search`. Also holds `FuseRanked` — the sorted-slice wrapper over `ReciprocalRankFusion` that most hybrid-search callers want.

- **Low-level `Scorer`/`DocStats`** (`bm25.go`): for bring-your-own-tokenizer, streaming, or scoring against a **per-query candidate set** whose IDF reflects only those candidates (the original file-search-on use case). Flow: `Tokenize` (or your own) → `StatsForTokens(tokens, queryTerms)` per doc → `NewScorer(queryTerms, docs, k1, b)` once → `Score(doc)` per doc.

### Key invariants — preserve these when editing scoring

- **IDF is the non-negative BM25+ variant**: `ln(1 + (N − n + 0.5)/(n + 0.5))`. This keeps a term appearing in every document at ~0 instead of going negative. Don't revert to the classic `ln((N − n + 0.5)/(n + 0.5))` form.
- **`DocStats` is query-scoped**: `TermFreqs` only holds counts for query terms, so it stays tiny regardless of body size. Keep it that way.
- **Same tokenizer for queries and documents** so they share a vocabulary. `Tokenize` lowercases maximal Unicode letter/digit runs; it mirrors file-search-on's `internal/fingerprint` tokenizer so keyword and near-duplicate views agree on word boundaries.
- **Parameter fallbacks happen at score time** in `NewScorer`: `k1 <= 0` → `DefaultK1`, `b` outside `[0,1]` → `DefaultB`, RRF `k <= 0` → 60. `NewWithParams` stores raw values unchecked.
- **Stable ordering**: `Search` and `FuseRanked` break score ties by ascending ID.

The doc comments in `bm25.go` carry the math and the design rationale (candidate-set IDF, issue #335) — read them before changing formulas.
