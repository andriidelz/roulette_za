package admin

import (
	"html/template"
	"time"
)

var tplFuncMap = template.FuncMap{
	"abs": func(x float64) float64 {
		if x < 0 {
			return -x
		}
		return x
	},
	"add": func(a, b int) int {
		return a + b
	},
	"subtract": func(a, b int) int {
		return a - b
	},
	"divide": func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return float64(a) / float64(b)
	},
	"multiply": func(a, b float64) float64 {
		return a * b
	},
	"formatDate": func(t time.Time) string {
		return t.Format("02.01.2006 15:04")
	},
	// Функция seq генерирует последовательность чисел от start до end включительно
	// Используется для создания диапазона страниц в пагинации
	"seq": func(start, end int) []int {
		if end < start {
			return []int{}
		}
		seq := make([]int, end-start+1)
		for i := range seq {
			seq[i] = start + i
		}
		return seq
	},
}
