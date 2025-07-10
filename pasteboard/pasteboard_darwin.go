//go:build darwin

package pasteboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

unsigned int clipboard_read_image(void **out);
char* clipboard_get_type();
char* clipboard_read_text();
char* clipboard_read_file_url();
char* clipboard_read_all_file_urls();
unsigned int clipboard_read_data(void **out);
*/
import "C"

import (
	"unsafe"
)

func initialize() error { return nil }

func readText() string {
	cText := C.clipboard_read_text()
	if cText == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cText))
	return C.GoString(cText)
}

func readImage() (buf []byte, err error) {
	var (
		data unsafe.Pointer
		n    C.uint
	)
	// nolint:gocritic
	n = C.clipboard_read_image(&data)
	if data == nil {
		return nil, errUnavailable
	}
	defer C.free(data)
	if n == 0 {
		return nil, nil
	}
	return C.GoBytes(data, C.int(n)), nil
}

func readFileURL() string {
	cURL := C.clipboard_read_file_url()
	if cURL == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cURL))
	return C.GoString(cURL)
}

func readData() (buf []byte, err error) {
	var (
		data unsafe.Pointer
		n    C.uint
	)
	// nolint:gocritic
	n = C.clipboard_read_data(&data)
	if data == nil {
		return nil, errUnavailable
	}
	defer C.free(data)
	if n == 0 {
		return nil, nil
	}
	return C.GoBytes(data, C.int(n)), nil
}

func getType() string {
	cType := C.clipboard_get_type()
	if cType == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cType))
	return C.GoString(cType)
}

func readAllFileURLs() string {
	cURLs := C.clipboard_read_all_file_urls()
	if cURLs == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cURLs))
	return C.GoString(cURLs)
}
