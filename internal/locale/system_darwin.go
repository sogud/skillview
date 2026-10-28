package locale

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
static int sv_language(int index, char *buffer, int length) {
    CFArrayRef languages = CFLocaleCopyPreferredLanguages();
    int found = languages && index < CFArrayGetCount(languages);
    if (found) {
        CFStringRef language = (CFStringRef)CFArrayGetValueAtIndex(languages, index);
        CFStringGetCString(language, buffer, length, kCFStringEncodingUTF8);
    }
    if (languages) CFRelease(languages);
    return found;
}
*/
import "C"

func SystemLanguage() string {
	var tags []string
	for index := 0; index < 64; index++ {
		var buffer [128]C.char
		if C.sv_language(C.int(index), &buffer[0], C.int(len(buffer))) == 0 {
			break
		}
		tags = append(tags, C.GoString(&buffer[0]))
	}
	return Preferred(tags)
}
