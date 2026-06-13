package subtitle

import "testing"

func TestGuessLang(t *testing.T) {
	tests := []struct {
		name string
		dirs []string
		file string
		want string
	}{
		// Real Miboujin nikki layout: language subfolder + language baked into name.
		{"chi subfolder", []string{"CHI"}, "Miboujin_nikki_[31BEFB6A].chi_Maho.sub.ass", "chi"},
		{"eng subfolder", []string{"ENG"}, "Miboujin_nikki_[31BEFB6A].eng_SubDESU-H.ass", "eng"},
		{"kor subfolder", []string{"KOR"}, "Miboujin_nikki_[31BEFB6A].kor.ass", "kor"},
		{"nested rus subfolder", []string{"RUS", "SUB"}, "Miboujin_nikki_[31BEFB6A].ass", "rus"},
		// Filename only (manual move into a flat folder), no directory hint.
		{"lang in filename", nil, "Show_[crc].fre_Johnny-subs.ass", "fre"},
		{"english full word", nil, "Show.S01E01.english.srt", "eng"},
		// Directory full name and 2-letter code.
		{"full name dir", []string{"French"}, "whatever.srt", "fre"},
		{"two letter dir", []string{"de"}, "whatever.srt", "ger"},
		// Two-letter tokens inside filenames must NOT match (false-positive guard).
		{"no false positive", nil, "It_Is_A_Show.srt", "und"},
		{"unrecognised", nil, "random.srt", "und"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GuessLang(tt.dirs, tt.file); got != tt.want {
				t.Errorf("GuessLang(%v, %q) = %q, want %q", tt.dirs, tt.file, got, tt.want)
			}
		})
	}
}
