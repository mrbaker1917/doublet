package game

import (
	"strings"
	"testing"
)

func testHintDictionary(t *testing.T) Dictionary {
	t.Helper()
	dict, err := LoadDictionaryFromReader(strings.NewReader("cat\ncot\ncab\ndog\n"))
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	return dict
}

func TestHintNextStepUsesSolutionPath(t *testing.T) {
	dict := testHintDictionary(t)
	solution := []string{"cat", "cot", "dog"}

	step, ok := HintNextStep(dict, "cat", "dog", solution)
	if !ok || step != "cot" {
		t.Fatalf("HintNextStep(cat) = %q, %v; want cot, true", step, ok)
	}

	step, ok = HintNextStep(dict, "cot", "dog", solution)
	if !ok || step != "dog" {
		t.Fatalf("HintNextStep(cot) = %q, %v; want dog, true", step, ok)
	}
}

func TestHintNextStepFallsBackToBFS(t *testing.T) {
	dict, err := LoadDictionaryFromReader(strings.NewReader("cat\ncot\ncog\ndog\ncab\n"))
	if err != nil {
		t.Fatalf("load dictionary: %v", err)
	}
	solution := []string{"cat", "cot", "cog", "dog"}

	step, ok := HintNextStep(dict, "cab", "dog", solution)
	if !ok || step != "cat" {
		t.Fatalf("HintNextStep(cab) = %q, %v; want cat, true", step, ok)
	}
}

func TestHintNextStepAtTarget(t *testing.T) {
	dict := testHintDictionary(t)
	solution := []string{"cat", "cot", "dog"}

	if step, ok := HintNextStep(dict, "dog", "dog", solution); ok {
		t.Fatalf("HintNextStep at target = %q, want false", step)
	}
}
