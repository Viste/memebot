package utils

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	reMdPre     = regexp.MustCompile("(?s)```(?:[a-zA-Z0-9_+\\-]*\\n)?(.*?)```")
	reMdCode    = regexp.MustCompile("`([^`\n]+)`")
	reMdBold    = regexp.MustCompile(`(?s)\*\*(.+?)\*\*`)
	reMdBoldAlt = regexp.MustCompile(`(?s)__(.+?)__`)
	reMdStrike  = regexp.MustCompile(`(?s)~~(.+?)~~`)
	reMdHeading = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s+(.+?)\s*#*\s*$`)
	reMdLink    = regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\s]+)\)`)
)

// MarkdownToTelegramHTML конвертирует стандартный markdown в подмножество HTML, поддерживаемое Telegram.
func MarkdownToTelegramHTML(s string) string {
	type span struct {
		kind    string // "pre" | "code"
		content string
	}
	var spans []span

	s = reMdPre.ReplaceAllStringFunc(s, func(m string) string {
		sub := reMdPre.FindStringSubmatch(m)
		spans = append(spans, span{"pre", strings.TrimRight(sub[1], "\n")})
		return fmt.Sprintf("\x00X%d\x00", len(spans)-1)
	})

	s = reMdCode.ReplaceAllStringFunc(s, func(m string) string {
		sub := reMdCode.FindStringSubmatch(m)
		spans = append(spans, span{"code", sub[1]})
		return fmt.Sprintf("\x00X%d\x00", len(spans)-1)
	})

	s = html.EscapeString(s)

	s = reMdBold.ReplaceAllString(s, "<b>$1</b>")
	s = reMdBoldAlt.ReplaceAllString(s, "<b>$1</b>")
	s = reMdStrike.ReplaceAllString(s, "<s>$1</s>")
	s = reMdHeading.ReplaceAllString(s, "<b>$1</b>")
	s = reMdLink.ReplaceAllString(s, `<a href="$2">$1</a>`)

	for i, sp := range spans {
		marker := fmt.Sprintf("\x00X%d\x00", i)
		var replacement string
		if sp.kind == "pre" {
			replacement = "<pre>" + html.EscapeString(sp.content) + "</pre>"
		} else {
			replacement = "<code>" + html.EscapeString(sp.content) + "</code>"
		}
		s = strings.Replace(s, marker, replacement, 1)
	}

	return s
}

// StripMarkdown убирает markdown-разметку, оставляя только текст
func StripMarkdown(s string) string {
	s = reMdPre.ReplaceAllString(s, "$1")
	s = reMdCode.ReplaceAllString(s, "$1")
	s = reMdBold.ReplaceAllString(s, "$1")
	s = reMdBoldAlt.ReplaceAllString(s, "$1")
	s = reMdStrike.ReplaceAllString(s, "$1")
	s = reMdHeading.ReplaceAllString(s, "$1")
	s = reMdLink.ReplaceAllString(s, "$1")
	return s
}
