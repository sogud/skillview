package library

/*
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>
int skillview_archive_move(const char *source, const char *destination);
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func macArchiveMove(source, destination string) error {
	from, to := C.CString(source), C.CString(destination)
	defer C.free(unsafe.Pointer(from))
	defer C.free(unsafe.Pointer(to))
	if C.skillview_archive_move(from, to) == 0 {
		return fmt.Errorf("macOS could not move the archived Skill between volumes")
	}
	return nil
}
