package abus

/*
#cgo pkg-config: ibus-1.0
#cgo CFLAGS: -std=gnu99
#cgo LDFLAGS: -lX11 -lXtst -pthread
#include <ibus.h>
#include <stdlib.h>

extern void initIBus();
extern char* randomFnc();
*/
import "C"
import (
	"unsafe"
)

func RandomFnc() string {
	var wmClass = C.randomFnc()
	if wmClass != nil {
		defer C.free(unsafe.Pointer(wmClass))
		return C.GoString(wmClass)
	}
	return ""
}

func InitIBus() {
	C.initIBus()
}
