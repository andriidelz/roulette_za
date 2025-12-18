package utils

type Lang string

const (
	RU Lang = "ru"
	UA Lang = "ua"
	EN Lang = "en"
)

type Forms struct {
	// Для RU/UA: one/few/many. Для EN используется one/many.
	One  string
	Few  string
	Many string
}

func idxRuUa(n int) int {
	if n < 0 {
		n = -n
	}
	if n%100 >= 11 && n%100 <= 14 {
		return 2
	}
	switch n % 10 {
	case 1:
		return 0
	case 2, 3, 4:
		return 1
	default:
		return 2
	}
}

func Plural(lang Lang, n int, f Forms) string {
	switch lang {
	case RU, UA:
		switch idxRuUa(n) {
		case 0:
			return f.One
		case 1:
			return f.Few
		default:
			return f.Many
		}
	default: // EN is default
		if n == 1 || n == -1 {
			return f.One
		}
		// for EN use Many as plural fallback; if empty, fall back to One+"s"
		if f.Many != "" {
			return f.Many
		}
		return f.One + "s"
	}
}

var lexicon = map[string]map[Lang]Forms{
	"points": {
		RU: {One: "балл", Few: "балла", Many: "баллов"},
		UA: {One: "бал", Few: "бали", Many: "балів"},
		EN: {One: "point", Many: "points"},
	},
}

func PluralWord(lang Lang, n int, key string) string {
	langs, ok := lexicon[key]
	if !ok {
		// unknown key: safe fallback
		return key
	}

	forms, ok := langs[lang]
	if !ok {
		forms, ok = langs[EN] // default EN
		if !ok {
			// any available
			for _, v := range langs {
				forms = v
				break
			}
		}
	}

	return Plural(lang, n, forms)
}

// func FuncMap() map[string]any {
// 	return map[string]any{
// 		"plural": func(lang string, n int, key string) string {
// 			return Word(Lang(lang), n, key)
// 		},
// 	}
// }

// {{.N}} {{plural .Lang .N "points"}}
