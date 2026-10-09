package library

/*
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>
char *skillview_trash(const char *path);
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func macTrash(path string) (string, error) {
	input := C.CString(path)
	defer C.free(unsafe.Pointer(input))
	output := C.skillview_trash(input)
	if output == nil {
		return "", fmt.Errorf("macOS could not move the item to Trash")
	}
	defer C.free(unsafe.Pointer(output))
	return C.GoString(output), nil
}
