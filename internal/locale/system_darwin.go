package locale

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
static void sv_language(char *buffer, int length) {
    CFArrayRef languages = CFLocaleCopyPreferredLanguages();
    if (languages && CFArrayGetCount(languages) > 0) {
        CFStringRef language = (CFStringRef)CFArrayGetValueAtIndex(languages, 0);
        CFStringGetCString(language, buffer, length, kCFStringEncodingUTF8);
    }
    if (languages) CFRelease(languages);
}
*/
import "C"

import "strings"

func SystemLanguage() string {
	var buffer [128]C.char
	C.sv_language(&buffer[0], C.int(len(buffer)))
	if strings.HasPrefix(C.GoString(&buffer[0]), "zh") {
		return "zh-CN"
	}
	return "en"
}
