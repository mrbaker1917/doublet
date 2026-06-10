package game

// HintNextStep returns the next word on a shortest path from current to end.
// It prefers the precomputed solution when current is still on that path.
func HintNextStep(dict Dictionary, current, end string, solution []string) (string, bool) {
	if current == end {
		return "", false
	}
	if step, ok := nextStepOnPath(solution, current); ok {
		return step, true
	}
	path, found := ShortestPathBFS(dict, current, end, 0)
	if !found || len(path) < 2 {
		return "", false
	}
	return path[1], true
}

func nextStepOnPath(path []string, current string) (string, bool) {
	for i := 0; i < len(path)-1; i++ {
		if path[i] == current {
			return path[i+1], true
		}
	}
	return "", false
}
