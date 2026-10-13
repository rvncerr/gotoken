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
- **Unicode normalization** — Input is normalized to NFC before segmentation, so canonically equivalent spellings (`é` vs. `e` + combining accent) yield identical sub-tokens; the form is configurable
- **Case folding** — Unicode full case folding after normalization, so `PDF` and `pdf` hash to the same feature; can be disabled
- **Concurrency-safe** — A `Tokenizer` is immutable once built and safe to share across goroutines, provided its `Policy` is too (all built-in policies are)
- **Minimal dependencies** — [`gocircular`](https://github.com/rvncerr/gocircular) for its generic ring buffer and [`golang.org/x/text`](https://pkg.go.dev/golang.org/x/text/unicode/norm) for normalization

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

## Normalization and Case Folding

Unicode allows the same text to be encoded in more than one way: `é` can be the single code
point U+00E9 or `e` followed by U+0301 COMBINING ACUTE ACCENT. Hashed indexes cannot tell these
apart after the fact, and the decomposed form would even be split at the accent, which is not a
letter. `Tokenize` therefore normalizes its input to
[NFC](https://www.unicode.org/reports/tr15/) before doing anything else, so both spellings of
`café` produce the same map:

```go
tok := gotoken.New(gotoken.WithLanguages(unicode.Latin))

tok.Tokenize("café")  // {"café": {Language: 0, Base: [0, 5]}}
tok.Tokenize("café") // {"café": {Language: 0, Base: [0, 5]}} — identical
```

The form is configurable. `NFKC` additionally folds compatibility variants (fullwidth letters,
ligatures, superscript digits) onto their plain counterparts, which widens matching at the cost
of conflating characters Unicode treats as distinct; because normalization runs on the whole
source, a no-break space then becomes a token boundary too. Callers that already normalize
upstream can skip the pass:

```go
import "golang.org/x/text/unicode/norm"

gotoken.New(gotoken.WithNormalization(norm.NFKC)) // fold compatibility variants
gotoken.New(gotoken.WithoutNormalization())       // input is already in the desired form
```

After normalization the text is case-folded with Unicode default case folding
([`cases.Fold`](https://pkg.go.dev/golang.org/x/text/cases#Fold)), so `PDF`, `Pdf`, and `pdf`
produce the same sub-tokens and hash to the same feature. This is *full* folding, defined for
matching rather than display: `Straße` and `STRASSE` both become `strasse`, and Greek final and
medial sigma fold to one letter. Disable it for a case-sensitive index:

```go
gotoken.New(gotoken.WithoutCaseFolding()) // keep "PDF" and "pdf" distinct
```

All sub-tokens and byte offsets in the result refer to the normalized, case-folded text.

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
| `WithNormalization(f norm.Form)` | Set the Unicode normalization form applied to input (default: `norm.NFC`) |
| `WithoutNormalization()` | Skip normalization; input is used as given |
| `WithoutCaseFolding()` | Skip case folding; `PDF` and `pdf` stay distinct |

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
