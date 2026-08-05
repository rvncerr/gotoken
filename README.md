# gotoken

A smart tokenization library for Go with multilingual support and language detection.

## Features

- **Sub-token generation** — Generates multiple overlapping tokens from compound words
- **Language detection** — Identifies language of text segments using Unicode range tables
- **Configurable depth** — Policy-based control over tokenization granularity
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
    // Create tokenizer with Latin and Cyrillic language support
    tok := gotoken.New(
        gotoken.WithLanguages(unicode.Latin, unicode.Cyrillic),
        gotoken.WithPolicy(gotoken.NewLinearPolicy(10, 10, 18, 2)),
    )

    // Tokenize multilingual text
    tokens := tok.Tokenize("hello мир")
    
    for token, info := range tokens {
        fmt.Printf("%q: language=%d, base=%v\n", token, info.Language, info.Base)
    }
}
```

## Configuration

### Policies

Control tokenization depth based on segment count:

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
- Guarantees bounded output size
- No partially filled depth levels
- O(1) depth calculation via closed-form solution

### Language Support

Add Unicode range tables for language detection:

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

## Token metadata

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

## API

### Types

- `Tokenizer` — Main tokenizer struct
- `TokenInfo` — Contains detected language index and byte offsets (see [Token metadata](#token-metadata))
- `Policy` — Interface for depth calculation
- `CountPolicy` — Limits total subtoken count (recommended)
- `LinearPolicy` — Interpolates depth linearly by segment count
- `FixedPolicy` — Returns constant depth
- `RuneClass` — Character classification (Letter, Digit, Punct, Other)

### Functions

- `New(opts ...Option) *Tokenizer` — Create a new tokenizer
- `WithPolicy(p Policy) Option` — Set the depth policy
- `WithLanguage(rt *unicode.RangeTable) Option` — Add a language
- `WithLanguages(tables ...*unicode.RangeTable) Option` — Add multiple languages

### Methods

- `(*Tokenizer) Tokenize(source string) map[string]TokenInfo` — Tokenize a string

## License

MIT
