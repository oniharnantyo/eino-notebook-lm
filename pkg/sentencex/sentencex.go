package sentencex

import (
	"github.com/wikimedia/sentencex-go/languages"
)

// Segment segments text into sentences using the specified language rules.
func Segment(language, text string) []string {
	factory := languages.LanguageFactory{}
	lang := factory.CreateLanguage(language)
	return lang.Segment(text)
}
