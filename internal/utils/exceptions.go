package utils

import (
	"context"
	"fmt"
	"runtime"
)

func PanicHandler(cancel context.CancelCauseFunc) {
	if r := recover(); r != nil {
		cancel(fmt.Errorf("%v", r))
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, false)
		fmt.Printf("Recovered from panic: %v\n%s\n", r, buf[:n])
	}
}
