//go:build darwin

package pasteboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

unsigned int clipboard_read_image(void **out);
*/
import "C"

import (
	"unsafe"
)

func initialize() error { return nil }

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
