package main

import (
	"context"
	"fmt"
	"github.com/pt-main/scholion/lang"
	"github.com/pt-main/scholion/lang/lcproc"
	"github.com/pt-main/scholion/lang/processing"

	"github.com/pt-main/lc/v2/engine/core"
)

func main() {
	l, err := lcproc.NewSyntax("//", "/*", "*/")
	fmt.Println(err)
	p := processing.NewProcessor()
	p.Disp.AddPattern(`\s*test (?<value>.*)?`, func(p *processing.Processor, m map[string]string) core.ErrorInterface {
		fmt.Println(m)
		return nil
	})
	e := lang.NewEngine(l, context.Background(), p)
	fmt.Println(e.Process(`
test code
//# test with args
	`))
	fmt.Println(p.GetCode())
}
