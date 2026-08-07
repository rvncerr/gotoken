# gotoken

A smart tokenization library for Go 1.25+ with multilingual support and language detection.

gotoken implements **SmartToken**: it splits each word at Unicode character-class and script
boundaries into segments, then emits every contiguous run of up to *D* segments as an
overlapping sub-token, where the depth *D* is chosen by a pluggable policy. The whole token is
always emitted, so exact-match lookups keep full recall.

## Features

- **Sub-token generation** — Generates multiple overlapping tokens from compound words
- **Language detection** — Identifies language of text segments using Unicode range tables
- **Configurable depth** — Policy-based control over tokenization granularity, including a hard per-token output cap
- **Concurrency-safe** — A `Tokenizer` is immutable once built and safe to share across goroutines, provided its `Policy` is too (all built-in policies are)
- **Minimal dependencies** — A single small dependency ([`gocircular`](https://github.com/rvncerr/gocircular)) for its generic ring buffer

## Installation

```bash
go get github.com/rvncerr/gotoken
```

## Usage

```go
package main

import (
    "fmt"
    "unicode"

    "github.com/rvncerr/gotoken"
)

func main() {
    // Create a tokenizer with Latin and Cyrillic language support.
    tok := gotoken.New(
        gotoken.WithLanguages(unicode.Latin, unicode.Cyrillic),
        gotoken.WithPolicy(gotoken.NewCountPolicy(16)),
    )

    tokens := tok.Tokenize("img2pdf helloпривет")

    for sub, info := range tokens {
        fmt.Printf("%q: language=%d, base=%v\n", sub, info.Language, info.Base)
    }
    // One map entry per distinct sub-token:
    //   "img", "2", "pdf", "img2", "2pdf", "img2pdf",
    //   "hello" (Latin), "привет" (Cyrillic), "helloпривет" (Cyrillic)
    // A query for "pdf" now matches a document containing "img2pdf".
}
```

## Policies

A policy maps a token's segment count to a tokenization depth; a depth ≤ 0 means "whole token
only". If no policy is configured, `NewLinearPolicy(10, 10, 18, 2)` is used.

```go
// Fixed depth for all tokens
gotoken.NewFixedPolicy(5)

// Linear interpolation based on segment count
// Short tokens (≤3 segments): depth 10
// Long tokens (≥10 segments): depth 2
// In between: linearly interpolated
gotoken.NewLinearPolicy(3, 10, 10, 2)

// Limit maximum number of subtokens (recommended)
// Algorithmically optimal: finds largest depth that stays within limit
// Priority: whole token → single segments → pairs → triples → ...
gotoken.NewCountPolicy(50)
```

**CountPolicy** is recommended for production use:

- Guarantees bounded output size (the whole token is charged against the limit)
- No partially filled depth levels
- O(1) depth calculation via closed-form solution

## Language Support

Add Unicode range tables for language detection. The order of added languages determines their
index (0, 1, 2, ...); the first matching table wins.

```go
tok := gotoken.New(
    gotoken.WithLanguage(unicode.Latin),
    gotoken.WithLanguage(unicode.Cyrillic),
    gotoken.WithLanguage(unicode.Han),
)

// Or add multiple at once
tok := gotoken.New(
    gotoken.WithLanguages(unicode.Latin, unicode.Cyrillic, unicode.Han),
)
```

## Token Metadata

Each entry in the returned map carries a `TokenInfo`:

```go
type TokenInfo struct {
    Language int    // index into the configured languages, or -1
    Base     [2]int // byte offsets [start, end) of the language-relevant portion
}
```

`Base` is relative to the start of the sub-token, not the source string. The
zero value `[0, 0]` is a **sentinel meaning "undetermined"** rather than an
empty span at offset 0 — check for it before slicing:

```go
tokens := tok.Tokenize("hello123 123...")

tokens["hello123"] // {Language: 0, Base: [0, 5]}  -> "hello" is the base
tokens["123..."]   // {Language: -1, Base: [0, 0]} -> no language-relevant portion
```

A token is undetermined when it contains no letters, or when it spans more than
three segments and so has no single identifiable language portion. This holds
uniformly, including under a policy that emits only whole tokens.

## API Reference

### Creation

| Function | Description |
|----------|-------------|
| `New(opts ...Option)` | Create a new tokenizer |
| `WithPolicy(p Policy)` | Set the depth policy (default: `NewLinearPolicy(10, 10, 18, 2)`) |
| `WithLanguage(rt *unicode.RangeTable)` | Add a language range table |
| `WithLanguages(tables ...*unicode.RangeTable)` | Add multiple language range tables |

### Policies

| Constructor | Behavior |
|-------------|----------|
| `NewFixedPolicy(depth)` | Constant depth for every token |
| `NewLinearPolicy(shortLen, shortDepth, longLen, longDepth)` | Interpolates depth between two anchors by segment count |
| `NewCountPolicy(maxCount)` | Largest depth that keeps a token within `maxCount` map entries (recommended) |

### Tokenization

| Method | Description |
|--------|-------------|
| `Tokenize(source string)` | Returns `map[string]TokenInfo`, keyed by sub-token text |

### Types

| Type | Description |
|------|-------------|
| `Tokenizer` | The tokenizer; immutable and safe for concurrent use |
| `TokenInfo` | Detected language index and base byte range (see [Token Metadata](#token-metadata)) |
| `Policy` | Interface mapping segment count to depth |
| `RuneClass` | Character classification (Letter, Digit, Punct, Other) |

## License

MIT License - see [LICENSE](LICENSE) file.
