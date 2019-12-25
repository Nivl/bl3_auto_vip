package blcodes

import "fmt"

// Printer is an interface used to print data back to the user
type Printer interface {
	Start(action string)
	Success()
	SuccessMsg(msg string)
	Failed(errMesg string)
	Info(msg string)
}

// we make sure stdPrinter implements Printer
var _ Printer = (*stdPrinter)(nil)

type stdPrinter struct {
}

func (p *stdPrinter) Start(action string) {
	fmt.Print(action + "... ")
}

func (p *stdPrinter) Success() {
	p.SuccessMsg(p.green("done!"))
}

func (p *stdPrinter) SuccessMsg(msg string) {
	fmt.Println(p.green(msg))
}

func (p *stdPrinter) Failed(errMsg string) {
	fmt.Println(p.red(errMsg))
}

func (p *stdPrinter) Info(msg string) {
	fmt.Println(msg)
}

func (p *stdPrinter) green(msg string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", msg)
}

func (p *stdPrinter) red(msg string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", msg)
}

// NewPrinter returns a printer that writes on stdout
func NewPrinter() Printer {
	return &stdPrinter{}
}
