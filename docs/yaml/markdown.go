package main

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	// mdHeading matches MarkDown H1..h6 headings. Note that this regex may produce
	// false positives for (e.g.) comments in code-blocks (# this is a comment),
	// so should not be used as a generic regex for other purposes.
	mdHeading = regexp.MustCompile(`^([#]{1,6})\s(.*)$`)
	// htmlAnchor matches inline HTML anchors. This is intended to only match anchors
	// for our use-case; DO NOT consider using this as a generic regex, or at least
	// not before reading https://stackoverflow.com/a/1732454/1811501.
	htmlAnchor = regexp.MustCompile(`<a\s+(?:name|id)="?([^"]+)"?\s*></a>\s*`)
)

// getSections returns all H2 sections by title (lowercase)
func getSections(mdString string) map[string]string {
	parsedContent := strings.Split("\n"+mdString, "\n## ")
	sections := make(map[string]string, len(parsedContent))
	for _, s := range parsedContent {
		if strings.HasPrefix(s, "#") {
			// not a H2 Section
			continue
		}
		parts := strings.SplitN(s, "\n", 2)
		if len(parts) == 2 {
			sections[strings.ToLower(parts[0])] = parts[1]
		}
	}
	return sections
}

// cleanupMarkDown cleans up the MarkDown passed in mdString for inclusion in
// YAML. It removes trailing whitespace and substitutes tabs for four spaces
// to prevent YAML switching to use "compact" form; ("line1  \nline\t2\n")
// which, although equivalent, is hard to read.
func cleanupMarkDown(mdString string) (md string, anchors []string) {
	// remove leading/trailing whitespace, and replace tabs in the whole content
	mdString = strings.TrimSpace(mdString)
	mdString = strings.ReplaceAll(mdString, "\t", "    ")
	mdString = strings.ReplaceAll(mdString, "https://docs.docker.com", "")

	var id string
	// replace trailing whitespace per line, and handle custom anchors
	lines := strings.Split(mdString, "\n")
	for i := 0; i < len(lines); i++ {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		lines[i], id = convertHTMLAnchor(lines[i])
		if id != "" {
			anchors = append(anchors, id)
		}
	}
	return strings.Join(lines, "\n"), anchors
}

// convertHTMLAnchor converts inline anchor-tags in headings (<a name=myanchor></a>)
// to an extended-markdown property ({#myanchor}). Extended Markdown properties
// are not supported in GitHub Flavored Markdown, but are supported by Jekyll,
// and lead to cleaner HTML in our docs, and prevents duplicate anchors.
// It returns the converted MarkDown heading and the custom ID (if present)
func convertHTMLAnchor(mdLine string) (md string, customID string) {
	if m := mdHeading.FindStringSubmatch(mdLine); len(m) > 0 {
		if a := htmlAnchor.FindStringSubmatch(m[2]); len(a) > 0 {
			customID = a[1]
			mdLine = m[1] + " " + htmlAnchor.ReplaceAllString(m[2], "") + " {#" + customID + "}"
		}
	}
	return mdLine, customID
}
