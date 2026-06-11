package textmatch

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Café!":            "cafe",
		"３×３ EYES":         "3 3 eyes",
		"  Attack on Titan ": "attack on titan",
		"Fate/Zero":        "fate zero",
		"Re:ZERO":          "re zero",
		"":                 "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSimilarityNorm(t *testing.T) {
	exact := SimilarityNorm(Normalize("Attack on Titan"), Normalize("Attack on Titan"))
	if exact != 100 {
		t.Errorf("exact match = %v, want 100", exact)
	}

	typo := SimilarityNorm(Normalize("atack on titn"), Normalize("attack on titan"))
	if typo < 70 {
		t.Errorf("one-char typo similarity = %v, want >= 70", typo)
	}

	reordered := SimilarityNorm(Normalize("titan attack on"), Normalize("attack on titan"))
	if reordered < 80 {
		t.Errorf("reordered words similarity = %v, want >= 80", reordered)
	}

	unrelated := SimilarityNorm(Normalize("naruto"), Normalize("attack on titan"))
	if unrelated > 40 {
		t.Errorf("unrelated similarity = %v, want <= 40", unrelated)
	}
}
