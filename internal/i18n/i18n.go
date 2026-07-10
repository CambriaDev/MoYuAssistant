package i18n

// UseEnglish controls whether the UI should prefer English text.
// It is toggled during app startup based on CJK font availability.
var UseEnglish bool

// T returns the English copy when English mode is enabled; otherwise it keeps
// the original Chinese string as the default UI text.
func T(zh, en string) string {
	if UseEnglish {
		return en
	}
	return zh
}
