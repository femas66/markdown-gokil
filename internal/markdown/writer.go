package markdown

import (
	"strings"

	"femas66/markdown-gokil/internal/docx"
)

// FromDocument converts a parsed docx Document into a Markdown string
func FromDocument(doc *docx.Document) (string, error) {
	var sb strings.Builder

	for i, p := range doc.Paragraphs {
		// Keep trailing spaces but trim right to handle empty lines naturally
		text := strings.TrimRight(p, " \t")
		sb.WriteString(text)

		// Add newline
		if i < len(doc.Paragraphs)-1 {
			// Apply Markdown soft break (two spaces + newline) for non-list/non-table text
			trimmed := strings.TrimSpace(text)
			isList := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || 
				(len(trimmed) > 2 && trimmed[1:3] == ". ") || (len(trimmed) > 3 && trimmed[2:4] == ". ")
			isTable := strings.HasPrefix(trimmed, "|")

			if len(text) > 0 && !isList && !isTable {
				sb.WriteString("  \n")
			} else {
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
