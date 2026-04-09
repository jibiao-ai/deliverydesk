// Package skill provides document parsing utilities for the skill engine.
package skill

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jibiao-ai/deliverydesk/pkg/logger"
)

// ParseDocument extracts text content from a document file.
// Supported formats: .docx, .xlsx, .txt, .md
func ParseDocument(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".docx":
		return parseDocx(filePath)
	case ".xlsx":
		return parseXlsx(filePath)
	case ".txt", ".md":
		return parseTextFile(filePath)
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

	// Then read each sheet
	var sb strings.Builder
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			text := extractSheetText(rc, sharedStrings)
			rc.Close()
			if text != "" {
				sb.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", filepath.Base(f.Name)))
				sb.WriteString(text)
				sb.WriteString("\n\n")
			}
		}
	}
	return sb.String(), nil
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
	var cellType string
	var rowStarted bool

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
			}
		case xml.EndElement:
			if t.Name.Local == "v" {
				inValue = false
			}
		case xml.CharData:
			if inValue {
				val := string(t)
				if cellType == "s" {
					// shared string reference
					idx := 0
					fmt.Sscanf(val, "%d", &idx)
					if idx >= 0 && idx < len(sharedStrings) {
						val = sharedStrings[idx]
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
