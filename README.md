# gotoken

A smart tokenization library for Go with multilingual support and language detection.

## Features

- **Sub-token generation** — Generates multiple overlapping tokens from compound words
- **Language detection** — Identifies language of text segments using Unicode range tables
- **Configurable depth** — Policy-based control over tokenization granularity
- **Zero dependencies** — Pure Go, no external packages required
- **Generics** — Type-safe implementation using Go 1.21+ generics

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

## API

### Types

- `Tokenizer` — Main tokenizer struct
- `TokenInfo` — Contains detected language index and byte offsets
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
