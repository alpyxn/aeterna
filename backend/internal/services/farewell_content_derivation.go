package services

import (
	"html"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var markdownRawHTMLPattern = regexp.MustCompile(`(?is)<\/?[a-z][a-z0-9-]*(\s[^>]*)?>`)
var markdownUnsafeLinkPattern = regexp.MustCompile(`(?i)\]\(\s*(javascript|vbscript|data):`)

var htmlToTextNormalizer = strings.NewReplacer(
	"</p>", " ",
	"<br>", " ",
	"<br/>", " ",
	"<br />", " ",
	"</li>", " ",
	"</h1>", " ",
	"</h2>", " ",
	"</h3>", " ",
	"</h4>", " ",
	"</h5>", " ",
	"</h6>", " ",
	"</blockquote>", " ",
	"</pre>", " ",
	"</code>", " ",
)

// sanitizeFarewellMarkdown removes inline raw HTML and neutralizes unsafe link schemes.
// HTML sanitization is still enforced again at render time.
func sanitizeFarewellMarkdown(md string) string {
	withoutRawHTML := markdownRawHTMLPattern.ReplaceAllString(md, "")
	return markdownUnsafeLinkPattern.ReplaceAllString(withoutRawHTML, "](#")
}

func markdownToPlainText(md string) string {
	if strings.TrimSpace(md) == "" {
		return ""
	}

	rendered := markdownToHTML(md)
	return htmlToPlainText(rendered)
}

func htmlToPlainText(rendered string) string {
	if strings.TrimSpace(rendered) == "" {
		return ""
	}

	rendered = htmlToTextNormalizer.Replace(rendered)
	withoutTags := htmlTagPattern.ReplaceAllString(rendered, " ")
	decoded := html.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(decoded), " ")
}

func countWordsFromMarkdown(md string) int {
	return len(strings.Fields(markdownToPlainText(md)))
}
