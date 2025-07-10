package main

import (
	"bytes"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// ExtractPDFFromString extracts a PDF document from a string and returns the PDF bytes
// and the surrounding text. The PDF is identified by "%PDF-" start marker and "%%EOF" end marker.
// Returns: pdfBytes, surroundingText, error
func ExtractPDFFromString(input string) ([]byte, string) {
	startMarker := "%PDF-"
	endMarker := "%%EOF"

	// Find the start of the PDF
	startIndex := strings.Index(input, startMarker)
	if startIndex == -1 {
		return nil, input
	}

	// Find the end of the PDF
	endIndex := strings.Index(input[startIndex:], endMarker)
	if endIndex == -1 {
		return nil, input
	}

	// Adjust endIndex to be relative to the original string
	endIndex += startIndex + len(endMarker)

	// Extract the PDF content
	pdfContent := input[startIndex:endIndex]

	// Extract the surrounding text (before and after the PDF)
	beforePDF := input[:startIndex]
	afterPDF := input[endIndex:]
	surroundingText := beforePDF + " " + afterPDF

	return []byte(pdfContent), surroundingText
}

// extractMultiplePDFsFromString extracts multiple PDF documents from a string and returns
// an array of PDF bytes and the surrounding text. The PDFs are identified by "%PDF-" start
// marker and "%%EOF" end marker pairs. All normal text between markers is concatenated
// and returned as surroundingText.
// Returns: [][]byte (array of PDF bytes), surroundingText
func ExtractMultiplePDFsFromString(input string) ([][]byte, string) {
	startMarker := "%PDF-"
	endMarker := "%%EOF"

	var pdfBytes [][]byte
	var surroundingTextParts []string

	searchStart := 0

	for {
		// Find the start of the next PDF
		startIndex := strings.Index(input[searchStart:], startMarker)
		if startIndex == -1 {
			// No more PDFs found, add remaining text
			if searchStart < len(input) {
				surroundingTextParts = append(surroundingTextParts, input[searchStart:])
			}
			break
		}

		// Adjust startIndex to be relative to the original string
		startIndex += searchStart

		// Add text before this PDF to surrounding text
		if startIndex > searchStart {
			surroundingTextParts = append(surroundingTextParts, input[searchStart:startIndex])
		}

		// Find the next PDF start marker to limit our search for the end marker
		nextStartIndex := strings.Index(input[startIndex+len(startMarker):], startMarker)
		var searchLimit int
		if nextStartIndex == -1 {
			// No next PDF, search to end of string
			searchLimit = len(input)
		} else {
			// Next PDF found, limit search to before it
			searchLimit = startIndex + len(startMarker) + nextStartIndex
		}

		// Find the end of this PDF within the limited search area
		// Use LastIndex to find the last %%EOF marker for this PDF
		// Before: endIndex := strings.Index(input[startIndex:searchLimit], endMarker)
		endIndex := strings.LastIndex(input[startIndex:searchLimit], endMarker)
		if endIndex == -1 {
			// No end marker found within the valid range, add remaining text and break
			surroundingTextParts = append(surroundingTextParts, input[startIndex:])
			break
		}

		// Adjust endIndex to be relative to the original string
		endIndex += startIndex + len(endMarker)

		// Extract the PDF content
		pdfContent := input[startIndex:endIndex]
		pdfBytes = append(pdfBytes, []byte(pdfContent))

		// Continue searching after this PDF
		searchStart = endIndex
	}

	// Join all surrounding text parts
	surroundingText := strings.Join(surroundingTextParts, " ")

	return pdfBytes, surroundingText
}

// validatePDF validates if the provided byte slice is a valid PDF document
// using the pdfcpu library's validate function
func validatePDF(pdfBytes []byte) error {
	reader := bytes.NewReader(pdfBytes)
	return api.Validate(reader, nil)
}
