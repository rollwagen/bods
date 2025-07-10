package main

import (
	"errors"
	"fmt"

	"github.com/rollwagen/bods/pasteboard"
)

type PasteboardContentHandler interface {
	CanHandle(contentType string) bool
	Handle(contentType string, data []byte) ([]Content, error)
}

type PasteboardProcessor struct {
	handlers []PasteboardContentHandler
	config   *Config
}

func NewPasteboardProcessor(config *Config) *PasteboardProcessor {
	processor := &PasteboardProcessor{config: config}

	// Register default handlers
	processor.AddHandler(&ImageContentHandler{config: config})
	processor.AddHandler(&FileURLContentHandler{config: config})
	processor.AddHandler(&TextContentHandler{config: config})
	processor.AddHandler(&PDFContentHandler{config: config})

	return processor
}

func (p *PasteboardProcessor) AddHandler(handler PasteboardContentHandler) {
	p.handlers = append(p.handlers, handler)
}

func (p *PasteboardProcessor) ProcessPasteboard() ([]Content, error) {
	contentType := pasteboard.GetContentType()
	logger.Printf("pasteboard type=%s", contentType)

	data := pasteboard.ReadData()
	if data == nil {
		return nil, errors.New("could not read data from pasteboard")
	}

	for _, handler := range p.handlers {
		if handler.CanHandle(contentType) {
			return handler.Handle(contentType, data)
		}
	}

	return nil, fmt.Errorf("unsupported pasteboard content type: %s", contentType)
}
