package main

import (
	"bytes"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// extractPDFFromString extracts a PDF document from a string and returns the PDF bytes
// and the surrounding text. The PDF is identified by "%PDF-" start marker and "%%EOF" end marker.
// Returns: pdfBytes, surroundingText, error
// TODO: handle multiple PDFs i.e. multiple start/end markers in string
func extractPDFFromString(input string) ([]byte, string) {
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

// validatePDF validates if the provided byte slice is a valid PDF document
// using the pdfcpu library's validate function
func validatePDF(pdfBytes []byte) error {
	reader := bytes.NewReader(pdfBytes)
	return api.Validate(reader, nil)
}
