// +build openbsd,cgo

package term

import (
	"unsafe"
)

/*
#include <unistd.h>
#include <termios.h>
#include <sys/ioctl.h>

// Small wrapper to get rid of variadic args of ioctl()
int set_winsize(int fd, struct winsize *ws) {
	return ioctl(fd, TIOCGWINSZ, ws);
}

int get_winsize(int fd, struct winsize *ws) {
    return ioctl(fd, TIOCSWINSZ, ws);
}
*/
import "C"

// GetWinsize returns the window size based on the specified file descriptor.
func GetWinsize(fd uintptr) (*Winsize, error) {
	ws := &Winsize{}
	ret, err := C.get_winsize(C.int(fd), (*C.struct_winsize)(unsafe.Pointer(ws)))
	// Skip retval = 0
	if ret == 0 {
		return ws, nil
	}
	return ws, err
}

// SetWinsize tries to set the specified window size for the specified file descriptor.
func SetWinsize(fd uintptr, ws *Winsize) error {
	ret, err := C.set_winsize(C.int(fd), (*C.struct_winsize)(unsafe.Pointer(ws)))
	// Skip retval = 0
	if ret == 0 {
		return nil
	}
	return err
}
