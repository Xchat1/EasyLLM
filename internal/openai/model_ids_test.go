package openai

import (
	"reflect"
	"testing"
)

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

func TestGPT6CodexModelIDs(t *testing.T) {
	got := GPT6CodexModelIDs()
	want := []string{
		CodexModelGPT6Astra,
		CodexModelGPT6Luna,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GPT6CodexModelIDs() = %#v, want %#v", got, want)
	}
}

func TestSupportedCodexModelIDs(t *testing.T) {
	got := SupportedCodexModelIDs()
	want := []string{
		CodexModelGPT6Astra,
		CodexModelGPT6Luna,
		CodexModelGPT56Sol,
		CodexModelGPT56Terra,
		CodexModelGPT56Luna,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedCodexModelIDs() = %#v, want %#v", got, want)
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

func TestPreferredCodexModelOrder(t *testing.T) {
	got := PreferredCodexModelOrder()
	want := []string{
		CodexModelGPT6Astra,
		CodexModelGPT6Luna,
		CodexModelGPT56Sol,
		CodexModelGPT56,
		CodexModelGPT56Terra,
		CodexModelGPT56Luna,
		CodexModelGPT54,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PreferredCodexModelOrder() = %#v, want %#v", got, want)
	}
}

func TestIsSupportedCodexModel(t *testing.T) {
	for _, id := range []string{
		CodexModelGPT6Astra,
		CodexModelGPT6Luna,
		CodexModelGPT56Sol,
		CodexModelGPT56Terra,
		CodexModelGPT56Luna,
		CodexModelGPT54,
		CodexModelGPT56,
	} {
		if !IsSupportedCodexModel(id) {
			t.Errorf("expected %q to be recognized as supported", id)
		}
	}

	if IsSupportedCodexModel("gpt-5.5") {
		t.Errorf("gpt-5.5 should no longer be marked as supported codex model")
	}

	if IsSupportedCodexModel("gpt-3.5-turbo") {
		t.Errorf("gpt-3.5-turbo should not be marked as supported codex model")
	}
}
