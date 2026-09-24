package openai

const (
	CodexModelGPT6Astra  = "gpt-6-astra"
	CodexModelGPT6Luna   = "gpt-6-luna"
	CodexModelGPT56      = "gpt-5.6"
	CodexModelGPT56Sol   = "gpt-5.6-sol"
	CodexModelGPT56Terra = "gpt-5.6-terra"
	CodexModelGPT56Luna  = "gpt-5.6-luna"
	CodexModelGPT54      = "gpt-5.4"
)

const CodexDefaultModel = CodexModelGPT56Sol

var codexGPT6ModelIDs = []string{
	CodexModelGPT6Astra,
	CodexModelGPT6Luna,
}

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

var codexSupportedModelIDs = []string{
	CodexModelGPT6Astra,
	CodexModelGPT6Luna,
	CodexModelGPT56Sol,
	CodexModelGPT56Terra,
	CodexModelGPT56Luna,
}

var codexPreferredModelOrder = []string{
	CodexModelGPT6Astra,
	CodexModelGPT6Luna,
	CodexModelGPT56Sol,
	CodexModelGPT56,
	CodexModelGPT56Terra,
	CodexModelGPT56Luna,
	CodexModelGPT54,
}

func GPT6CodexModelIDs() []string {
	return append([]string(nil), codexGPT6ModelIDs...)
}

// GPT-5.6 Sol Pro is a reasoning mode, not a separate API model ID.
func GPT56CodexModelIDs() []string {
	return append([]string(nil), codexGPT56ModelIDs...)
}

func GPT56CompatibleCodexModelIDs() []string {
	return append([]string(nil), codexGPT56CompatibleModelIDs...)
}

func SupportedCodexModelIDs() []string {
	return append([]string(nil), codexSupportedModelIDs...)
}

func PreferredCodexModelOrder() []string {
	return append([]string(nil), codexPreferredModelOrder...)
}

func IsSupportedCodexModel(id string) bool {
	switch id {
	case CodexModelGPT6Astra, CodexModelGPT6Luna,
		CodexModelGPT56, CodexModelGPT56Sol, CodexModelGPT56Terra, CodexModelGPT56Luna,
		CodexModelGPT54:
		return true
	default:
		return false
	}
}
