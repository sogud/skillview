package library

/*
#include <fcntl.h>
*/
import "C"

// x/sys/unix does not expose Darwin's search-only directory flag.
const directorySearchFlag = int(C.O_SEARCH)
