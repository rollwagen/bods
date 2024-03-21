//go:build darwin

//
// See https://developer.apple.com/documentation/appkit/nspasteboard?language=objc
//

#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

unsigned int clipboard_read_image(void **out) {
	NSPasteboard * pasteboard = [NSPasteboard generalPasteboard];
	NSData *data = [pasteboard dataForType:NSPasteboardTypePNG];
	if (data == nil) {
		return 0;
	}
	NSUInteger size = [data length];
	*out = malloc(size);
	[data getBytes: *out length: size];
	return size;
}
