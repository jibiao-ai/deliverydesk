// Package skill provides document parsing utilities for the skill engine.
package skill

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// ParseDocument extracts text content from a document file.
// Supported formats: .docx, .xlsx, .txt, .md, .csv
func ParseDocument(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".docx":
		return parseDocx(filePath)
	case ".xlsx":
		return parseXlsx(filePath)
	case ".txt", ".md":
		return parseTextFile(filePath)
	case ".csv":
		return parseCsv(filePath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// parseDocx extracts text from a .docx file (OpenXML format)
func parseDocx(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	var sb strings.Builder
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open document.xml: %w", err)
			}
			text, err := extractDocxText(rc)
			rc.Close()
			if err != nil {
				return "", err
			}
			sb.WriteString(text)
		}
	}
	return sb.String(), nil
}

func extractDocxText(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder
	var inText bool
	var inParagraph bool

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sb.String(), nil // return what we have on parse error
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "t":
				inText = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inParagraph {
					sb.WriteString("\n")
					inParagraph = false
				}
			case "t":
				inText = false
			}
		case xml.CharData:
			if inText {
				sb.Write(t)
			}
		}
	}
	return sb.String(), nil
}

// parseXlsx extracts text from a .xlsx file (OpenXML spreadsheet)
func parseXlsx(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open xlsx: %w", err)
	}
	defer r.Close()

	// First read shared strings
	sharedStrings := parseSharedStrings(r)
	logger.Log.Infof("parseXlsx: %s loaded %d shared strings", filepath.Base(filePath), len(sharedStrings))

	// Then read each sheet
	var sb strings.Builder
	sheetCount := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				logger.Log.Warnf("parseXlsx: failed to open sheet %s: %v", f.Name, err)
				continue
			}
			text := extractSheetText(rc, sharedStrings)
			rc.Close()
			if text != "" {
				sheetCount++
				sb.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", filepath.Base(f.Name)))
				sb.WriteString(text)
				sb.WriteString("\n\n")
			}
		}
	}
	result := sb.String()
	logger.Log.Infof("parseXlsx: %s parsed %d sheets, total content length=%d bytes",
		filepath.Base(filePath), sheetCount, len(result))

	// Validate: if we got shared strings but the output looks like pure numbers,
	// it means shared string resolution failed — log a warning
	if len(sharedStrings) > 0 && len(result) > 0 {
		cjkCount := 0
		for _, r := range result {
			if isCJK(r) || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				cjkCount++
			}
		}
		ratio := float64(cjkCount) / float64(len([]rune(result)))
		if ratio < 0.1 {
			logger.Log.Warnf("parseXlsx: %s has very low text content ratio (%.1f%%), shared strings may not have resolved correctly",
				filepath.Base(filePath), ratio*100)
		}
	}

	return result, nil
}

func parseSharedStrings(r *zip.ReadCloser) []string {
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return nil
			}
			defer rc.Close()
			return extractSharedStrings(rc)
		}
	}
	return nil
}

func extractSharedStrings(r io.Reader) []string {
	decoder := xml.NewDecoder(r)
	var strings_list []string
	var inT bool
	var current strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inT = true
			}
			if t.Name.Local == "si" {
				current.Reset()
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inT = false
			}
			if t.Name.Local == "si" {
				strings_list = append(strings_list, current.String())
			}
		case xml.CharData:
			if inT {
				current.Write(t)
			}
		}
	}
	return strings_list
}

func extractSheetText(r io.Reader, sharedStrings []string) string {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder
	var inValue bool
	var inInlineStr bool
	var inInlineT bool
	var cellType string
	var rowStarted bool
	var inlineText strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				if rowStarted {
					sb.WriteString("\n")
				}
				rowStarted = true
			case "c":
				cellType = ""
				for _, attr := range t.Attr {
					if attr.Name.Local == "t" {
						cellType = attr.Value
					}
				}
			case "v":
				inValue = true
			case "is":
				// Inline string element — some xlsx files use <is><t>text</t></is>
				// instead of shared strings
				inInlineStr = true
				inlineText.Reset()
			case "t":
				if inInlineStr {
					inInlineT = true
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inValue = false
			case "t":
				inInlineT = false
			case "is":
				// End of inline string — write the collected text
				if inInlineStr {
					val := inlineText.String()
					if val != "" {
						if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
							sb.WriteString("\t")
						}
						sb.WriteString(val)
					}
					inInlineStr = false
				}
			}
		case xml.CharData:
			if inInlineT {
				inlineText.Write(t)
			} else if inValue {
				val := string(t)
				if cellType == "s" {
					// shared string reference
					idx := 0
					fmt.Sscanf(val, "%d", &idx)
					if idx >= 0 && idx < len(sharedStrings) {
						val = sharedStrings[idx]
					} else {
						// Shared string index out of range — skip this cell
						// to avoid inserting meaningless numeric indices
						logger.Log.Debugf("extractSheetText: shared string index %d out of range (have %d strings), skipping cell",
							idx, len(sharedStrings))
						continue
					}
				}
				if sb.Len() > 0 && !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString("\t")
				}
				sb.WriteString(val)
			}
		}
	}
	if rowStarted {
		sb.WriteString("\n")
	}
	return sb.String()
}

func parseTextFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

// parseCsv extracts text from a .csv file, converting rows into readable text
func parseCsv(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	// Allow variable number of fields per record
	reader.FieldsPerRecord = -1

	var sb strings.Builder
	var headers []string
	rowNum := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Log.Warnf("parseCsv: error reading row %d: %v", rowNum, err)
			continue
		}

		if rowNum == 0 {
			// First row is headers
			headers = record
			sb.WriteString(strings.Join(record, "\t"))
			sb.WriteString("\n")
		} else {
			// Data rows: if we have headers, format as "header: value" pairs
			// This improves semantic search quality
			if len(headers) > 0 {
				var parts []string
				for i, val := range record {
					val = strings.TrimSpace(val)
					if val == "" {
						continue
					}
					if i < len(headers) {
						parts = append(parts, fmt.Sprintf("%s: %s", headers[i], val))
					} else {
						parts = append(parts, val)
					}
				}
				if len(parts) > 0 {
					sb.WriteString(strings.Join(parts, ", "))
					sb.WriteString("\n")
				}
			} else {
				sb.WriteString(strings.Join(record, "\t"))
				sb.WriteString("\n")
			}
		}
		rowNum++
	}

	logger.Log.Infof("parseCsv: %s parsed %d rows", filepath.Base(filePath), rowNum)
	return sb.String(), nil
}

// IndexDocumentFile parses and indexes a document file for a skill
func IndexDocumentFile(skillID, docID uint, docName, filePath string) (int, error) {
	content, err := ParseDocument(filePath)
	if err != nil {
		return 0, fmt.Errorf("parse document %s: %w", docName, err)
	}

	if strings.TrimSpace(content) == "" {
		return 0, fmt.Errorf("document %s is empty after parsing", docName)
	}

	chunks := GetStore().IndexDocument(skillID, docID, docName, content)
	logger.Log.Infof("Indexed document %s for skill %d: %d chunks", docName, skillID, chunks)
	return chunks, nil
}
