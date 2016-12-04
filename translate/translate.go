// Package translate provides the Translater-class for lingualization, using JSON-mappings with github.com/Compufreak345/go-i18n/i18n
// & provides special functions for view-specific translation files.
package translate

import (
	"html/template"
	"strings"

	"github.com/Compufreak345/go-i18n/i18n"
)

// Translater provides functions for translating
type Translater struct {
	DefaultLang               string
	FallbackLang              string
	UrlLang                   string
	translationsByViewAndLang map[string]map[string]map[string]interface{}
}

const tTag = "goodl-lib/translate.go" //bl

type TFunc func(key string) string

// HTML translates a string to a template.HTML-object.
func (t *Translater) HTML(s string, args ...interface{}) template.HTML {
	return template.HTML(t.T(s, args...))
}

// T translates a string with the given arguments for i18n.TFunc.
func (t *Translater) T(key string, args ...interface{}) string {
	Tfunc, _ := i18n.Tfunc(t.DefaultLang, t.FallbackLang)
	return Tfunc(key, args...)
}

// MustLoadTranslationFile loads the given translation-file and panics if it is not found.
func (t *Translater) MustLoadTranslationFile(path string) {
	i18n.MustLoadTranslationFile(path)

}

// GetTranslationsForView gets the translations for the given view.
func (t *Translater) GetTranslationsForView(vName string) map[string]interface{} {
	if t.translationsByViewAndLang == nil {
		t.translationsByViewAndLang = make(map[string]map[string]map[string]interface{})
	}
	var trans map[string]interface{} = t.translationsByViewAndLang[t.DefaultLang][vName]
	if trans != nil {
		return trans
	}
	allTrans := i18n.GetBundle().GetTranslations()[strings.ToLower(t.DefaultLang)]

	viewTrans := make(map[string]interface{})

	//dbg.V(tTag, "Vname : %v, allTrans : %v,DefaultLang : %v, GetTranslations : %v", vName, allTrans, t.DefaultLang, i18n.GetBundle().GetTranslations())
	if allTrans != nil {
		vprefix := vName + "_"
		for k, trans := range allTrans {
			if strings.HasPrefix(k, vprefix) || strings.HasPrefix(k, "shared_") {
				viewTrans[k] = trans.MarshalInterface()
			}
		}
	}
	if t.translationsByViewAndLang[t.DefaultLang] == nil {
		t.translationsByViewAndLang[t.DefaultLang] = make(map[string]map[string]interface{})
	}
	t.translationsByViewAndLang[t.DefaultLang][vName] = viewTrans

	return viewTrans
}
