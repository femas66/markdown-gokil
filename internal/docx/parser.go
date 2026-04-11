package docx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func isValTrue(attrs []xml.Attr) bool {
	for _, attr := range attrs {
		if attr.Name.Local == "val" {
			v := strings.ToLower(attr.Value)
			if v == "0" || v == "false" || v == "off" {
				return false
			}
			return true
		}
	}
	return true
}

type Document struct {
	Paragraphs []string
}

type Relationship struct {
	Id     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

type Relationships struct {
	XMLName xml.Name       `xml:"Relationships"`
	Rels    []Relationship `xml:"Relationship"`
}

type Numbering struct {
	NumMap map[string]string            // numId -> abstractNumId
	Format map[string]map[int]string     // abstractNumId -> ilvl -> numFmt
}

func parseNumbering(archive *zip.ReadCloser) (*Numbering, error) {
	num := &Numbering{
		NumMap: make(map[string]string),
		Format: make(map[string]map[int]string),
	}

	var rc io.ReadCloser
	var err error
	for _, f := range archive.File {
		if f.Name == "word/numbering.xml" {
			rc, err = f.Open()
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if rc == nil {
		return num, nil // Missing numbering.xml is normal
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var currentAbstractId string
	var currentIlvl int

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "abstractNum":
				for _, attr := range se.Attr {
					if attr.Name.Local == "abstractNumId" {
						currentAbstractId = attr.Value
						num.Format[currentAbstractId] = make(map[int]string)
					}
				}
			case "lvl":
				for _, attr := range se.Attr {
					if attr.Name.Local == "ilvl" {
						val, _ := strconv.Atoi(attr.Value)
						currentIlvl = val
					}
				}
			case "numFmt":
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						num.Format[currentAbstractId][currentIlvl] = attr.Value
					}
				}
			case "num":
				var numId string
				for _, attr := range se.Attr {
					if attr.Name.Local == "numId" {
						numId = attr.Value
					}
				}
				// Map to abstractNumId
				for {
					t2, _ := decoder.Token()
					if t2 == nil { break }
					if ee, ok := t2.(xml.EndElement); ok && ee.Name.Local == "num" { break }
					if s2, ok := t2.(xml.StartElement); ok && s2.Name.Local == "abstractNumId" {
						for _, a2 := range s2.Attr {
							if a2.Name.Local == "val" {
								num.NumMap[numId] = a2.Value
							}
						}
					}
				}
			}
		}
	}

	return num, nil
}

// Parse extracts media, relations, and converts content to Markdown format
func Parse(docxPath, imageOutDir string) (*Document, error) {
	// 1. Unzip document (DOCX is a ZIP of XML files)
	r, err := zip.OpenReader(docxPath)
	if err != nil {
		if err == zip.ErrFormat {
			return nil, fmt.Errorf("invalid file format: %w", err)
		}
		return nil, fmt.Errorf("failed to open docx: %w", err)
	}
	defer r.Close()

	// 2. Extract images
	err = extractImages(r, imageOutDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract images: %w", err)
	}

	// 3. Map relations (document.xml.rels)
	rels, err := parseRels(r)
	if err != nil {
		return nil, fmt.Errorf("failed to map relations: %w", err)
	}

	// 4. Map numbering (numbering.xml)
	numbering, err := parseNumbering(r)
	if err != nil {
		return nil, fmt.Errorf("failed to map numbering: %w", err)
	}

	// 5. Parse main content (document.xml)
	var docXML io.ReadCloser
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			docXML, err = f.Open()
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if docXML == nil {
		return nil, fmt.Errorf("word/document.xml not found")
	}
	defer docXML.Close()

	paras, err := parseDocumentXML(docXML, rels, numbering)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	return &Document{Paragraphs: paras}, nil
}

func extractImages(archive *zip.ReadCloser, outDir string) error {
	dirCreated := false

	for _, f := range archive.File {
		// Look for media files
		if strings.HasPrefix(f.Name, "word/media/") && !f.FileInfo().IsDir() {
			// Create output directory if media exists
			if !dirCreated {
				if err := os.MkdirAll(outDir, 0755); err != nil {
					return err
				}
				dirCreated = true
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}

			// Copy with original name
			outFile := filepath.Join(outDir, filepath.Base(f.Name))
			wc, err := os.Create(outFile)
			if err != nil {
				rc.Close()
				return err
			}

			_, err = io.Copy(wc, rc)
			wc.Close()
			rc.Close()

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseRels(archive *zip.ReadCloser) (map[string]string, error) {
	relsMap := make(map[string]string)

	for _, f := range archive.File {
		if f.Name == "word/_rels/document.xml.rels" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()

			var data Relationships
			if err := xml.NewDecoder(rc).Decode(&data); err != nil {
				return nil, err
			}

			// Map Relation (ID -> Target)
			for _, r := range data.Rels {
				relsMap[r.Id] = r.Target
			}
			return relsMap, nil
		}
	}
	// Return empty map if no relations
	return relsMap, nil
}

func parseDocumentXML(r io.Reader, rels map[string]string, numbering *Numbering) ([]string, error) {
	decoder := xml.NewDecoder(r)
	var paras []string

	var currentParagraph strings.Builder
	var inText bool
	var formatBold bool
	var formatItalic bool

	// Calculate heading size per paragraph
	var maxFontSize float64
	var paraHasBold bool

	// List state
	listCounters := make(map[string]map[int]int)
	var lastNumId string
	var lastIlvl int = -1

	var paraNumId string
	var paraIlvl int = -1

	// Table state
	var inCell bool
	var tableRows [][]string
	var currentRow []string
	var currentCell strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch se := token.(type) {
		case xml.StartElement:
			tagName := se.Name.Local
			switch tagName {
			case "tbl":
				tableRows = nil
			case "tr":
				currentRow = nil
			case "tc":
				inCell = true
				currentCell.Reset()
			case "p":
				currentParagraph.Reset()
				maxFontSize = 0.0
				paraHasBold = false
				paraNumId = ""
				paraIlvl = -1
			case "pStyle":
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						styleVal := strings.ToLower(attr.Value)
						if strings.Contains(styleVal, "title") || strings.Contains(styleVal, "heading1") {
							if maxFontSize < 24 {
								maxFontSize = 24.0
							}
						} else if strings.Contains(styleVal, "heading2") {
							if maxFontSize < 18 {
								maxFontSize = 18.0
							}
						} else if strings.Contains(styleVal, "heading3") {
							if maxFontSize < 14 {
								maxFontSize = 14.0
							}
						}
					}
				}
			case "numId":
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						paraNumId = attr.Value
					}
				}
			case "ilvl":
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						val, err := strconv.Atoi(attr.Value)
						if err == nil {
							paraIlvl = val
						}
					}
				}
			case "r":
				formatBold = false
				formatItalic = false
			case "b", "bCs":
				// Check if bold
				if isValTrue(se.Attr) {
					formatBold = true
					paraHasBold = true
				}
			case "i", "iCs":
				// Check if italic
				if isValTrue(se.Attr) {
					formatItalic = true
				}
			case "sz", "szCs":
				for _, attr := range se.Attr {
					if attr.Name.Local == "val" {
						val, err := strconv.Atoi(attr.Value)
						if err == nil {
							// Value is in half-points
							ptSize := float64(val) / 2.0
							if ptSize > maxFontSize {
								maxFontSize = ptSize
							}
						}
					}
				}
			case "t":
				inText = true
			case "br":
				currentParagraph.WriteString("\n")
			case "blip":
				// Detect embedded image via rId
				for _, attr := range se.Attr {
					if attr.Name.Local == "embed" {
						relID := attr.Value
						// Match rId with relationship dictionary
						if target, ok := rels[relID]; ok {
							imageName := filepath.Base(target)
							// Insert as Markdown image syntax
							currentParagraph.WriteString(fmt.Sprintf("\n![image](images/%s)\n", imageName))
						}
					}
				}
			}
		case xml.CharData:
			if inText {
				text := string(se)
				trimmed := strings.TrimSpace(text)

				if (formatBold || formatItalic) && trimmed != "" {
					// Extract leading and trailing whitespace
					startIdx := strings.Index(text, trimmed)
					leading := text[:startIdx]
					trailing := text[startIdx+len(trimmed):]

					var marker string
					if formatBold && formatItalic {
						marker = "***"
					} else if formatBold {
						marker = "**"
					} else {
						marker = "*"
					}
					currentParagraph.WriteString(leading + marker + trimmed + marker + trailing)
				} else {
					currentParagraph.WriteString(text)
				}
			}
		case xml.EndElement:
			tagName := se.Name.Local
			switch tagName {
			case "tbl":
				if len(tableRows) > 0 {
					paras = append(paras, formatMarkdownTable(tableRows))
				}
			case "tr":
				tableRows = append(tableRows, currentRow)
			case "tc":
				inCell = false
				// Clean cell content
				cellText := strings.ReplaceAll(currentCell.String(), "\n", " ")
				currentRow = append(currentRow, cellText)
			case "p":
				text := currentParagraph.String()

				// Clean redundant markers from XML fragmentation
				text = strings.ReplaceAll(text, "******", "")
				text = strings.ReplaceAll(text, "****", "")

				// Skip empty paragraphs (not in cells)
				if strings.TrimSpace(text) == "" {
					if inCell {
						continue
					}
					// Only allow one empty paragraph to avoid excessive spacing
					if len(paras) > 0 && paras[len(paras)-1] != "" {
						paras = append(paras, "")
					}
					continue
				}

				if maxFontSize == 0 {
					maxFontSize = 11.0 
				}

				var prefix string
				if !inCell && paraNumId == "" {
					if maxFontSize >= 24 {
						prefix = "# "
					} else if maxFontSize >= 18 {
						prefix = "## "
					} else if maxFontSize >= 14 {
						prefix = "### "
					} else if maxFontSize >= 13 || (maxFontSize >= 12 && paraHasBold) {
						prefix = "#### "
					}
				}

				// Remove redundant bold if it's a heading
				if prefix != "" {
					if strings.HasPrefix(text, "***") && strings.HasSuffix(text, "***") {
						text = text[3 : len(text)-3]
					} else if strings.HasPrefix(text, "**") && strings.HasSuffix(text, "**") {
						text = text[2 : len(text)-2]
					}
				}

				fullText := prefix + text

				// Handle lists/numbering
				if paraNumId != "" {
					listPrefix := ""
					// Indent 2 spaces per level
					if paraIlvl > 0 {
						listPrefix = strings.Repeat("  ", paraIlvl)
					}

					marker := "- " // Default bullet
					if absId, ok := numbering.NumMap[paraNumId]; ok {
						if fmtStr, ok := numbering.Format[absId][paraIlvl]; ok {
							if listCounters[paraNumId] == nil {
								listCounters[paraNumId] = make(map[int]int)
							}

							// Reset deeper levels if we moved up or changed list
							if paraNumId != lastNumId || paraIlvl < lastIlvl {
								for l := range listCounters[paraNumId] {
									if l > paraIlvl {
										listCounters[paraNumId][l] = 0
									}
								}
							}

							listCounters[paraNumId][paraIlvl]++
							count := listCounters[paraNumId][paraIlvl]

							switch fmtStr {
							case "decimal":
								marker = fmt.Sprintf("%d. ", count)
							case "lowerLetter":
								marker = fmt.Sprintf("%c. ", 'a'+(count-1)%26)
							case "upperLetter":
								marker = fmt.Sprintf("%c. ", 'A'+(count-1)%26)
							case "lowerRoman":
								marker = fmt.Sprintf("%s. ", intToRoman(count, false))
							case "upperRoman":
								marker = fmt.Sprintf("%s. ", intToRoman(count, true))
							default:
								marker = "- "
							}
						}
					}
					lastNumId = paraNumId
					lastIlvl = paraIlvl
					fullText = listPrefix + marker + fullText
				}

				if inCell {
					if currentCell.Len() > 0 {
						currentCell.WriteString(" ")
					}
					currentCell.WriteString(fullText)
				} else {
					paras = append(paras, fullText)
				}
			case "t":
				inText = false
			}
		}
	}
	return paras, nil
}

func formatMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n") // Vertical space before table

	// Calculate max columns
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	for i, row := range rows {
		sb.WriteString("|")
		for j := 0; j < maxCols; j++ {
			cell := ""
			if j < len(row) {
				cell = strings.TrimSpace(row[j])
				// Escape pipes in cells
				cell = strings.ReplaceAll(cell, "|", "\\|")
			}
			sb.WriteString(" " + cell + " |")
		}
		sb.WriteString("\n")

		// Header separator after first row
		if i == 0 {
			sb.WriteString("|")
			for j := 0; j < maxCols; j++ {
				sb.WriteString(" --- |")
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func intToRoman(n int, upper bool) string {
	if n <= 0 {
		return ""
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	res := ""
	for i := 0; i < len(vals); i++ {
		for n >= vals[i] {
			res += syms[i]
			n -= vals[i]
		}
	}
	if !upper {
		return strings.ToLower(res)
	}
	return res
}
