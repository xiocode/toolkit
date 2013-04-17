package timelib

// #include "lib/timelib.h"

import "C"

import "unsafe"

type Datetime struct {
	time     *C.timelib_time
	time_now *C.timelib_time
}

//timelib_time *timelib_strtotime(char *s, int len,
//timelib_error_container **errors, const timelib_tzdb *tzdb, timelib_tz_get_wrapper tz_get_wrapper);
func Example() {
	err := *C.timelib_error_container
	test := "1988-01-26"
	cstime := C.CString(test)
	defer C.free(unsafe.Pointer(timestr))
	// datetime := Datetime{}
	C.timelib_strtotime(cstime, C.int(len(test)+1), *err, C.timelib_builtin_db())
}
