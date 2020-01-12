package main

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/russross/blackfriday"
)

// This file provides an balckfriday.v1 HtmlRenderer with custom
// extensions.

var htmlFlags = blackfriday.HTML_USE_SMARTYPANTS | blackfriday.HTML_SMARTYPANTS_DASHES

var markdownExtFlags = blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_HARD_LINE_BREAK |
	blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK |
	blackfriday.EXTENSION_HEADER_IDS |
	blackfriday.EXTENSION_TITLEBLOCK |
	blackfriday.EXTENSION_AUTO_HEADER_IDS |
	blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
	blackfriday.EXTENSION_DEFINITION_LISTS |
	blackfriday.EXTENSION_JOIN_LINES

// markdownHtmlRenderer is a markdown-to-HTML global renderer to use
// in templates.
var markdownHtmlRenderer = &htmlRenderer{
	Html: blackfriday.HtmlRenderer(htmlFlags, "", "").(*blackfriday.Html),
}

type htmlRenderer struct {
	*blackfriday.Html
}

// Image given ![alt](src) creates a <figure> with <img /> and
// <figcaption>alt</figcaption>.
func (r *htmlRenderer) Image(out *bytes.Buffer, link []byte, title []byte, alt []byte) {

	escapeSingleChar := func(char byte) (string, bool) {
		if char == '"' {
			return "&quot;", true
		}
		if char == '&' {
			return "&amp;", true
		}
		if char == '<' {
			return "&lt;", true
		}
		if char == '>' {
			return "&gt;", true
		}
		return "", false
	}

	attrEscape := func(out *bytes.Buffer, src []byte) {
		org := 0
		for i, ch := range src {
			if entity, ok := escapeSingleChar(ch); ok {
				if i > org {
					// copy all the normal characters since the last escape
					out.Write(src[org:i])
				}
				org = i + 1
				out.WriteString(entity)
			}
		}
		if org < len(src) {
			out.Write(src[org:])
		}
	}

	out.WriteString("<figure><img src=\"")
	//r.maybeWriteAbsolutePrefix(out, link)
	attrEscape(out, link)
	out.WriteString("\" alt=\"")
	if len(alt) > 0 {
		attrEscape(out, alt)
	}
	if len(title) > 0 {
		out.WriteString("\" title=\"")
		attrEscape(out, title)
	}

	out.WriteByte('"')
	out.WriteString(" />")
	out.WriteString(fmt.Sprintf("<figcaption>%s</figcaption>", alt))
	out.WriteString("</figure>")
}

func Markdown(s string) template.HTML {
	output := blackfriday.Markdown([]byte(s), markdownHtmlRenderer, markdownExtFlags)
	return template.HTML(string(output))
}
