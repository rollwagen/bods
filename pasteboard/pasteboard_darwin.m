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
		// Set out pointer to NULL so Go code can detect unavailable clipboard
		*out = NULL;
		return 0;
	}
	NSUInteger size = [data length];
	*out = malloc(size);
	[data getBytes: *out length: size];
	return size;
}

// Return the content type of the clipboard
// e.g. public.utf8-plain-text, public.png, public.file-url
char* clipboard_get_type() {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSArray *types = [pasteboard types];

	if ([types count] == 0) {
		return NULL;
	}

	NSString *firstType = [types objectAtIndex:0];
	const char *cString = [firstType UTF8String];

	if (cString == NULL) {
		return NULL;
	}

	size_t length = strlen(cString);
	char *result = malloc(length + 1);
	strcpy(result, cString);

	return result;
}

char* clipboard_read_text() {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSString *text = [pasteboard stringForType:NSPasteboardTypeString];

	if (text == nil) {
		return NULL;
	}

	const char *cString = [text UTF8String];
	if (cString == NULL) {
		return NULL;
	}

	size_t length = strlen(cString);
	char *result = malloc(length + 1);
	strcpy(result, cString);

	return result;
}

char* clipboard_read_file_url() {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSString *fileURLString = [pasteboard stringForType:NSPasteboardTypeFileURL];

	if (fileURLString == nil) {
		return NULL;
	}

	// Convert string to NSURL and resolve file reference URLs
	NSURL *url = [NSURL URLWithString:fileURLString];
	if (url == nil) {
		return NULL;
	}

	// Resolve file reference URLs to actual file paths
	NSString *resolvedPath = [url path];
	if (resolvedPath == nil) {
		return NULL;
	}

	// Return as proper file:// URL
	NSURL *resolvedURL = [NSURL fileURLWithPath:resolvedPath];
	NSString *result = [resolvedURL absoluteString];

	const char *cString = [result UTF8String];
	if (cString == NULL) {
		return NULL;
	}

	size_t length = strlen(cString);
	char *resultCString = malloc(length + 1);
	strcpy(resultCString, cString);

	return resultCString;
}

char* clipboard_read_all_file_urls() {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSArray *items = [pasteboard pasteboardItems];

	if ([items count] == 0) {
		return NULL;
	}

	NSMutableArray *fileURLs = [NSMutableArray array];

	// Iterate through all pasteboard items
	for (NSPasteboardItem *item in items) {
		// Check if the item has file URL data
		NSString *fileURLString = [item stringForType:NSPasteboardTypeFileURL];
		if (fileURLString != nil) {
			// Convert string to NSURL and resolve file reference URLs
			NSURL *url = [NSURL URLWithString:fileURLString];
			if (url != nil) {
				// Resolve file reference URLs to actual file paths
				NSString *resolvedPath = [url path];
				if (resolvedPath != nil) {
					// Convert back to proper file:// URL
					NSURL *resolvedURL = [NSURL fileURLWithPath:resolvedPath];
					NSString *urlString = [resolvedURL absoluteString];
					if (urlString != nil) {
						[fileURLs addObject:urlString];
					}
				}
			}
		}
	}

	if ([fileURLs count] == 0) {
		return NULL;
	}

	// Join all URLs with newlines (following text/uri-list format)
	NSString *joinedURLs = [fileURLs componentsJoinedByString:@"\n"];

	const char *cString = [joinedURLs UTF8String];
	if (cString == NULL) {
		return NULL;
	}

	size_t length = strlen(cString);
	char *resultCString = malloc(length + 1);
	strcpy(resultCString, cString);

	return resultCString;
}

unsigned int clipboard_read_data(void **out) {
	NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
	NSArray *types = [pasteboard types];

	if ([types count] == 0) {
		// Set out pointer to NULL so Go code can detect unavailable clipboard
		*out = NULL;
		return 0;
	}

	NSString *firstType = [types objectAtIndex:0];
	NSData *data = [pasteboard dataForType:firstType];

	if (data == nil) {
		// Set out pointer to NULL so Go code can detect unavailable clipboard
		*out = NULL;
		return 0;
	}

	NSUInteger size = [data length];
	*out = malloc(size);
	[data getBytes: *out length: size];
	return size;
}
