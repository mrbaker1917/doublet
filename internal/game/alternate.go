package game

// AnotherValidPath returns a path from start to end that differs from playerPath when possible.
func AnotherValidPath(dict Dictionary, start, end string, playerPath []string) ([]string, bool) {
	shortest, ok := ShortestPathBFS(dict, start, end, 0)
	if !ok {
		return nil, false
	}
	if !pathsEqual(shortest, playerPath) {
		return shortest, true
	}

	for i := 1; i < len(playerPath)-1; i++ {
		alt, ok := shortestPathAvoidingWord(dict, start, end, playerPath[i])
		if ok && !pathsEqual(alt, playerPath) {
			return alt, true
		}
	}

	minSteps := len(shortest) - 1
	for extra := 1; extra <= 3; extra++ {
		alt, ok := shortestPathAtLeast(dict, start, end, minSteps+extra)
		if ok && !pathsEqual(alt, playerPath) {
			return alt, true
		}
	}

	return nil, false
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func shortestPathAvoidingWord(dict Dictionary, start, end, blocked string) ([]string, bool) {
	if start == blocked || end == blocked {
		return nil, false
	}
	filtered := make(Dictionary, len(dict))
	for word := range dict {
		if word != blocked {
			filtered[word] = struct{}{}
		}
	}
	return ShortestPathBFS(filtered, start, end, 0)
}

func shortestPathAtLeast(dict Dictionary, start, end string, minChanges int) ([]string, bool) {
	if start == end {
		if minChanges <= 0 {
			return []string{start}, true
		}
		return nil, false
	}
	if len(start) != len(end) {
		return nil, false
	}

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

		if cur.word == end && cur.steps >= minChanges {
			return reconstructPath(prev, start, end), true
		}

		for _, nxt := range Neighbors(dict, cur.word) {
			nextSteps := cur.steps + 1
			if nxt == end && nextSteps < minChanges {
				continue
			}
			if visited[nxt] {
				continue
			}
			visited[nxt] = true
			prev[nxt] = cur.word
			queue = append(queue, node{word: nxt, steps: nextSteps})
		}
	}

	return nil, false
}

func reconstructPath(prev map[string]string, start, end string) []string {
	path := []string{end}
	for path[len(path)-1] != start {
		path = append(path, prev[path[len(path)-1]])
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
