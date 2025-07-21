package utils

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// https://core.telegram.org/bots/api#formatting-options
var (
	telegramTags = map[string]string{
		"b": "", "strong": "", // bold
		"i": "", "em": "", // italic
		"u": "", "ins": "", // underline
		"s": "", "strike": "", "del": "", // strikethrough
		"a":          "", // link
		"blockquote": "", // blockQuote

		"tg-emoji":   "",
		"tg-spoiler": "",
		"pre":        "",
	}
	paragraph = "\n\n"
)

// ParseNode - parse text as html, delete all useless tags
func ParseNode(text string, useTelMarkup bool) (string, error) {

	if text == "" {
		return text, nil
	}

	if !strings.Contains(text, "<") && !strings.Contains(text, "&lt") {
		// No HTML code
		return text, nil
	}

	result := ""
	htmlTokens := html.NewTokenizer(strings.NewReader(text))

loop:
	for {
		tokenType := htmlTokens.Next()
		if tokenType == html.ErrorToken {
			if htmlTokens.Err() == io.EOF {
				break loop
			}
			// If any other error - return it
			return "", htmlTokens.Err()
		}

		clearToken(htmlTokens.Token(), &result, useTelMarkup)
	}

	// remove leading and trailing white space
	// convert html symbols like &#34; to "
	return strings.TrimSpace(html.UnescapeString(result)), nil
}

// clearToken - parse text, normalize and delete all useless tags
func clearToken(t html.Token, result *string, useMarkup bool) {

	switch t.Type {
	case html.TextToken:
		// add text
		(*result) += t.String() // text + convert tags like - "<" - "&lt;"
		// result.Description += t.Data

	case html.StartTagToken:
		// add only available tags
		// example checking - t.Data == b || t.String() === <b>
		if useMarkup {
			if _, ok := telegramTags[t.Data]; ok {
				(*result) += t.String()
			}
		}
		if t.Data == "br" {
			// convert tag br to new paragraph
			(*result) += paragraph
		}

	case html.EndTagToken:
		if useMarkup {
			if _, ok := telegramTags[t.Data]; ok {
				(*result) += t.String()
			}
		}
		if t.Data == "p" || t.Data == "br" {
			// convert tags p and br to new paragraph
			(*result) += paragraph
		}
	}
}
