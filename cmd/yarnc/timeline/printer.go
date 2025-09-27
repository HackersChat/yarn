package timeline

import "fmt"

type printer interface {
	Print(format string, a ...interface{})
}

type defaultPrinter struct {
}

func (d defaultPrinter) Print(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}
