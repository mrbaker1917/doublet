package game

func OneLetterApart(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	diff := 0
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			diff++
			if diff > 1 {
				return false
			}
		}
	}
	return diff == 1
}

func Neighbors(dict Dictionary, word string) []string {
	out := make([]string, 0, 16)
	bytes := []byte(word)
	for i := 0; i < len(bytes); i++ {
		orig := bytes[i]
		for c := byte('a'); c <= byte('z'); c++ {
			if c == orig {
				continue
			}
			bytes[i] = c
			cand := string(bytes)
			if IsWord(dict, cand) {
				out = append(out, cand)
			}
		}
		bytes[i] = orig
	}
	return out
}

// ShortestPathBFS finds a word ladder with at most maxChanges transitions.
func ShortestPathBFS(dict Dictionary, start, end string, maxChanges int) ([]string, bool) {
	if start == end {
		return []string{start}, true
	}
	if maxChanges < 0 {
		return nil, false
	}
	if len(start) != len(end) {
		return nil, false
	}
	unlimited := maxChanges == 0

	type node struct {
		word  string
		steps int
	}

	queue := []node{{word: start, steps: 0}}
	visited := map[string]bool{start: true}
	prev := map[string]string{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if !unlimited && cur.steps >= maxChanges {
			continue
		}

		for _, nxt := range Neighbors(dict, cur.word) {
			if visited[nxt] {
				continue
			}
			visited[nxt] = true
			prev[nxt] = cur.word
			if nxt == end {
				path := []string{end}
				for path[len(path)-1] != start {
					path = append(path, prev[path[len(path)-1]])
				}
				for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
					path[i], path[j] = path[j], path[i]
				}
				return path, true
			}
			queue = append(queue, node{word: nxt, steps: cur.steps + 1})
		}
	}

	return nil, false
}
