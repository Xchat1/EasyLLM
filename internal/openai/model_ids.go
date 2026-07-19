package openai

const (
	CodexModelGPT56      = "gpt-5.6"
	CodexModelGPT56Sol   = "gpt-5.6-sol"
	CodexModelGPT56Terra = "gpt-5.6-terra"
	CodexModelGPT56Luna  = "gpt-5.6-luna"
	CodexModelGPT55      = "gpt-5.5"
	CodexModelGPT54      = "gpt-5.4"
)

const CodexDefaultModel = CodexModelGPT56Sol

var codexGPT56ModelIDs = []string{
	CodexModelGPT56Sol,
	CodexModelGPT56Terra,
	CodexModelGPT56Luna,
}

var codexGPT56CompatibleModelIDs = []string{
	CodexModelGPT56Sol,
	CodexModelGPT56,
	CodexModelGPT56Terra,
	CodexModelGPT56Luna,
}

// GPT-5.6 Sol Pro is a reasoning mode, not a separate API model ID.
func GPT56CodexModelIDs() []string {
	return append([]string(nil), codexGPT56ModelIDs...)
}

func GPT56CompatibleCodexModelIDs() []string {
	return append([]string(nil), codexGPT56CompatibleModelIDs...)
}
