package game

// IsPlayableWord reports whether a word is suitable for casual doublet play.
func IsPlayableWord(word string) bool {
	if len(word) < 3 {
		return false
	}
	hasVowel := false
	for i := 0; i < len(word); i++ {
		c := word[i]
		if c < 'a' || c > 'z' {
			return false
		}
		switch c {
		case 'a', 'e', 'i', 'o', 'u':
			hasVowel = true
		case 'q', 'x', 'z', 'j', 'y':
			return false
		}
	}
	return hasVowel
}
