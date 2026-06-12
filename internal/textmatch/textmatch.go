// Package textmatch provides shared title normalisation and similarity scoring
// used by both the indexer (torrent title matching) and the metadata service
// (anime title search).
package textmatch

import (
	"strings"
	"unicode"

	"github.com/adrg/strutil/metrics"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"golang.org/x/text/width"
)

// Normalize lowercases, folds full-width forms and diacritics to ASCII, keeps
// only letters and digits, and collapses every other run to a single space.
// e.g. "Café!" -> "cafe", "３×３ EYES" -> "3x3 eyes".
func Normalize(s string) string {
	s = width.Narrow.String(s)
	// A new chain is created per call: transform.Chain is stateful and not safe
	// for concurrent use, so sharing a package-level instance would be a data race
	// when multiple goroutines call Normalize simultaneously (e.g. metadata /search).
	stripper := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	if folded, _, err := transform.String(stripper, s); err == nil {
		s = folded
	}
	s = strings.ToLower(s)

	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
		} else if !prevSpace {
			b.WriteRune(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

var bigramDice = func() *metrics.SorensenDice {
	m := metrics.NewSorensenDice()
	m.NgramSize = 2
	return m
}()

// SimilarityNorm returns a 0–100 similarity between two already-normalised
// strings (see Normalize). It is the max of three signals: token-level
// Sørensen–Dice, a token overlap coefficient (when lengths are comparable), and
// character-bigram Sørensen–Dice.
func SimilarityNorm(na, nb string) float64 {
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 100
	}

	tA, tB := strings.Fields(na), strings.Fields(nb)
	freq := map[string]int{}
	for _, t := range tA {
		freq[t]++
	}
	inter := 0
	for _, t := range tB {
		if freq[t] > 0 {
			inter++
			freq[t]--
		}
	}
	minLen, maxLen := len(tA), len(tB)
	if minLen > maxLen {
		minLen, maxLen = maxLen, minLen
	}

	diceScore := float64(2*inter) / float64(len(tA)+len(tB)) * 100

	overlapScore := diceScore
	if maxLen > 0 && float64(minLen)/float64(maxLen) >= 0.6 {
		overlapScore = float64(inter) / float64(minLen) * 100
	}

	sa := strings.ReplaceAll(na, " ", "")
	sb := strings.ReplaceAll(nb, " ", "")
	charScore := bigramDice.Compare(sa, sb) * 100

	best := diceScore
	if overlapScore > best {
		best = overlapScore
	}
	if charScore > best {
		best = charScore
	}
	return best
}

// Similarity returns a 0–100 similarity between two raw (un-normalised) titles.
func Similarity(a, b string) float64 {
	return SimilarityNorm(Normalize(a), Normalize(b))
}
