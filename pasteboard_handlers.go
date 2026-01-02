package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// -------------------------------------------------------------------------

// ImageContentHandler handles image content from pasteboard
type ImageContentHandler struct {
	config *Config
}

func (h *ImageContentHandler) CanHandle(contentType string) bool {
	return contentType == MessageContentTypeMediaTypePNG ||
		contentType == MessageContentTypeMediaTypeGIF ||
		contentType == MessageContentTypeMediaTypeWEBP ||
		contentType == MessageContentTypeMediaTypeJPEG
}

func (h *ImageContentHandler) Handle(contentType string, data []byte) ([]Content, error) {
	if !IsVisionCapable(h.config.ModelID) {
		return nil, fmt.Errorf("%s: model does not have vision capability", h.config.ModelID)
	}

	imgType := http.DetectContentType(data)
	if !slices.Contains(MessageContentTypes, imgType) {
		return nil, fmt.Errorf("unsupported image type: %s", imgType)
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("could not decode image: %v", err)
	}

	isValidSize, msg := validateImageDimension(img, format)
	if !isValidSize {
		return nil, fmt.Errorf("invalid image size: %s", msg)
	}

	content := imgToMessageContent(data, imgType)
	return []Content{content}, nil
}

// -------------------------------------------------------------------------

// FileURLContentHandler handles file URLs from pasteboard
type FileURLContentHandler struct {
	config *Config
}

func (h *FileURLContentHandler) CanHandle(contentType string) bool {
	return contentType == "text/uri-list"
}

func (h *FileURLContentHandler) Handle(contentType string, data []byte) ([]Content, error) {
	urls := strings.Split(string(data), "\n")
	var contents []Content

	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" || strings.HasPrefix(url, "#") {
			continue // Skip empty lines and comments
		}

		content, err := h.processFileURL(url)
		if err != nil {
			logger.Printf("Error processing URL %s: %v", url, err)
			continue
		}

		if content != nil {
			contents = append(contents, *content)
		}
	}

	return contents, nil
}

func (h *FileURLContentHandler) processFileURL(url string) (*Content, error) {
	logger.Printf("processFileURL(url=%s)\n", url)
	// Handle file:// URLs
	if strings.HasPrefix(url, "file://") {
		filePath := strings.TrimPrefix(url, "file://")
		return h.processLocalFile(filePath)
	}

	// Handle HTTP/HTTPS URLs - could download and process
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return h.processRemoteURL(url)
	}

	return nil, fmt.Errorf("unsupported URL scheme: %s", url)
}

func (h *FileURLContentHandler) processLocalFile(filePath string) (*Content, error) {
	// Resolve any URL encoding
	decodedPath, err := url.QueryUnescape(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file path: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(decodedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", decodedPath)
	}

	// Determine file type and process accordingly
	file, err := os.Open(decodedPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(decodedPath))
	mimeType, _ := getContentType(file)

	logger.Printf("processLocalFile mimeType=%s ext=%s\n", mimeType, ext)

	switch mimeType {
	case "image/png", "image/jpg", "image/jpeg", "image/gif", "image/webp":
		return h.processImageFile(decodedPath)
	case "application/pdf":
		return h.processPDFFile(decodedPath)
	case "text/plain":
		return h.processTextFile(decodedPath)
	default:
		return h.processUnknownFile(decodedPath) // for unknown types, treat as text if small enough

	}
}

func (h *FileURLContentHandler) processImageFile(filePath string) (*Content, error) {
	if !IsVisionCapable(h.config.ModelID) {
		return nil, fmt.Errorf("model does not support vision capability")
	}

	imgBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	imgType := http.DetectContentType(imgBytes)
	if !slices.Contains(MessageContentTypes, imgType) {
		return nil, fmt.Errorf("unsupported image type: %s", imgType)
	}

	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	isValidSize, msg := validateImageDimension(img, format)
	if !isValidSize {
		return nil, fmt.Errorf("invalid image dimensions: %s", msg)
	}

	content := imgToMessageContent(imgBytes, imgType)
	return &content, nil
}

func (h *FileURLContentHandler) processPDFFile(filePath string) (*Content, error) {
	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF file: %v", err)
	}

	if err := validatePDF(pdfBytes); err != nil {
		return nil, fmt.Errorf("invalid PDF file: %v", err)
	}

	b64Pdf := base64.StdEncoding.EncodeToString(pdfBytes)
	source := Source{
		Type:      "base64",
		MediaType: MessageContentTypeMediaTypePDF,
		Data:      b64Pdf,
	}

	content := Content{
		Type:   MessageContentTypeDocument,
		Source: &source,
		Citations: &Citations{
			Enabled: false,
		},
	}

	if IsPromptCachingSupported(h.config.ModelID) {
		content.CacheControl = &CacheControl{Type: "ephemeral"}
	}

	return &content, nil
}

func (h *FileURLContentHandler) processTextFile(filePath string) (*Content, error) {
	textBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %v", err)
	}

	content := Content{
		Type: MessageContentTypeText,
		Text: fmt.Sprintf("Content from file %s:\n\n%s", filepath.Base(filePath),
			string(textBytes)),
	}

	if IsPromptCachingSupported(h.config.ModelID) {
		content.CacheControl = &CacheControl{Type: "ephemeral"}
	}

	return &content, nil
}

func (h *FileURLContentHandler) processRemoteURL(url string) (*Content, error) {
	// For now, just add the URL as text content
	// Could be extended to download and process the content
	content := Content{
		Type: MessageContentTypeText,
		Text: fmt.Sprintf("URL from pasteboard: %s", url),
	}

	return &content, nil
}

func (h *FileURLContentHandler) processUnknownFile(filePath string) (*Content, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contentType, _ := getContentType(file)
	logger.Printf("processUnknownFile contentType=%s\n", contentType)
	// Check file size - only process if reasonably small
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if info.Size() > 1024*1024 { // 1MB limit
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	return h.processTextFile(filePath)
}

// -------------------------------------------------------------------------

// TextContentHandler handles plain text from pasteboard
type TextContentHandler struct {
	config *Config
}

func (h *TextContentHandler) CanHandle(contentType string) bool {
	return contentType == "text/plain"
}

func (h *TextContentHandler) Handle(contentType string, data []byte) ([]Content, error) {
	content := Content{
		Type: MessageContentTypeText,
		Text: fmt.Sprintf("Text from pasteboard:\n\n%s", string(data)),
	}

	return []Content{content}, nil
}

// -------------------------------------------------------------------------

// PDFContentHandler handles PDF content from pasteboard
type PDFContentHandler struct {
	config *Config
}

func (h *PDFContentHandler) CanHandle(contentType string) bool {
	return contentType == MessageContentTypeMediaTypePDF
}

func (h *PDFContentHandler) Handle(contentType string, data []byte) ([]Content, error) {
	if err := validatePDF(data); err != nil {
		return nil, fmt.Errorf("invalid PDF content: %v", err)
	}

	b64Pdf := base64.StdEncoding.EncodeToString(data)
	source := Source{
		Type:      "base64",
		MediaType: MessageContentTypeMediaTypePDF,
		Data:      b64Pdf,
	}

	content := Content{
		Type:   MessageContentTypeDocument,
		Source: &source,
		Citations: &Citations{
			Enabled: false,
		},
	}

	if IsPromptCachingSupported(h.config.ModelID) {
		content.CacheControl = &CacheControl{Type: "ephemeral"}
	}

	return []Content{content}, nil
}

func getContentTypeFromString(input string) (string, error) {
	reader := strings.NewReader(input)
	return getContentType(reader)
}

// func getContentType(file *os.File) (string, error) {
func getContentType(seeker io.ReadSeeker) (string, error) {
	// file implements io.ReadSeeker
	// var seeker io.ReadSeeker = file

	// At most the first 512 bytes of data are used:
	// https://golang.org/src/net/http/sniff.go?s=646:688#L11
	buff := make([]byte, 512)

	_, err := seeker.Seek(0, io.SeekStart)
	if err != nil {
		return "", err
	}

	bytesRead, err := seeker.Read(buff)
	if err != nil && err != io.EOF {
		return "", err
	}

	// Slice to remove fill-up zero values which cause a wrong content type detection in the next step
	buff = buff[:bytesRead]

	return http.DetectContentType(buff), nil
}
