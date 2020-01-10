package main

import (
	"html/template"
	"strings"
	"unicode/utf8"

	"github.com/bahna/magazine/appkit"
	"github.com/bahna/magazine/appkit/tmplfuncs"
	"github.com/bahna/magazine/cms"
	"github.com/nicksnyder/go-i18n/i18n"
)

func generateTmplFuncs(app *appkit.Application) template.FuncMap {
	m := map[string]interface{}{
		"T":            i18n.IdentityTfunc,
		"langName":     tmplfuncs.LangTagName,
		"langCode":     tmplfuncs.LangTagStr,
		"idToStr":      tmplfuncs.ObjectIDToString,
		"plus":         tmplfuncs.Plus,
		"incr":         tmplfuncs.Increment,
		"hasRole":      tmplfuncs.HasRole,
		"fmtTime":      tmplfuncs.FmtTime,
		"fmtTimeShort": tmplfuncs.FmtTimeShort,
		"zeroTime":     tmplfuncs.ZeroTime,
		"pubDate":      tmplfuncs.PubDate,
		"fmtInputTime": tmplfuncs.FmtInputTime,
		"inputTimeNow": tmplfuncs.InputTimeNow,
		"hasID":        tmplfuncs.HasID,
		"md":           tmplfuncs.Markdown,
		"translit":     tmplfuncs.Translit(app),
		"joinUsers":    tmplfuncs.JoinUsers,
		"joinTopics":   tmplfuncs.JoinTopics,
		"srcset":       tmplfuncs.Srcset,
		"bytesToMb":    tmplfuncs.BytesToMb,
		"dayNumber":    tmplfuncs.DayNumber,
		"month":        tmplfuncs.Month,
		"monthShort":   tmplfuncs.MonthShort,

		"mapTopicsToStyles": mapTopicsToStyles,
		"cutLine":           cutLine,
	}
	return template.FuncMap(m)
}

func cutLine(s string, n int) template.HTML {
	rr := []rune{}
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		rr = append(rr, r)
		s = s[size:]
	}
	if len(rr) > n {
		s = strings.TrimSpace(string(rr[:n])) + "…"
	} else {
		s = string(rr)
	}
	return template.HTML(s)
}

// mapTopicsToStyles maps a topic to its unique CSS-style.
func mapTopicsToStyles(tt []*cms.Topic) map[string]string {
	var topicCardStyles = []string{
		"rounded-a-lot",
		"rounded-rb-corner-a-lot",
		"",
		"rounded-lb-corner-a-lot",
	}
	m := make(map[string]string)
	l := len(topicCardStyles)
	for i, t := range tt {
		m[t.Title] = topicCardStyles[i%l]
	}
	return m
}
