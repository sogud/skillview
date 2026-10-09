package library

/*
#include <copyfile.h>
#include <errno.h>
static int copy_metadata(int source, int destination) {
 if (fcopyfile(source, destination, NULL, COPYFILE_ACL | COPYFILE_XATTR) != 0) return errno;
 return 0;
}
*/
import "C"

import (
	"os"
	"syscall"
)

func copyMetadata(source, destination *os.File) error {
	code := C.copy_metadata(C.int(source.Fd()), C.int(destination.Fd()))
	if code != 0 {
		return syscall.Errno(code)
	}
	return nil
}
