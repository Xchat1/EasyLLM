package openai

import "testing"

func TestGPT56CodexModelIDsExposeThreeSeriesModels(t *testing.T) {
	got := GPT56CodexModelIDs()
	want := []string{
		CodexModelGPT56Sol,
		CodexModelGPT56Terra,
		CodexModelGPT56Luna,
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d model ids, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("model id %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestGPT56CompatibleCodexModelIDsIncludesAlias(t *testing.T) {
	got := GPT56CompatibleCodexModelIDs()
	found := false
	for _, id := range got {
		if id == CodexModelGPT56 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected compatibility ids to include %q: %#v", CodexModelGPT56, got)
	}
}
