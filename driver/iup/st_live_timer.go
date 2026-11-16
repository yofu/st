package stgui

/*
#include <stdint.h>
#include <iup.h>

static void stEnableTimerAction(uintptr_t handle) {
	Ihandle *timer = (Ihandle *)handle;
	IupSetCallback(timer, "ACTION_CB", IupGetCallback(timer, "ACTION"));
}
*/
import "C"

import "github.com/yofu/go-iup/iup"

// The bundled go-iup registers TimerAction under ACTION. IUP timers dispatch
// ACTION_CB, so install the same Go trampoline under the native callback name.
func enableTimerAction(timer *iup.Handle) {
	C.stEnableTimerAction(C.uintptr_t(timer.Native()))
}
