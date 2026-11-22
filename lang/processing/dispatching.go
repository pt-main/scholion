package processing

import (
	"github.com/pt-main/scholion/lang/lcproc"

	"github.com/dlclark/regexp2"
	"github.com/pt-main/lc/v2/engine/core"
)

func (d *Dispatch) AddPattern(pattern string, handler DirectiveFunc) (int, core.ErrorInterface) {
	for idx, patt := range d.sources {
		if patt == pattern {
			return idx, core.Err(lcproc.PatternError, "Has same pattern: %v", pattern)
		}
	}
	d.sources = append(d.sources, pattern)
	d.Functions = append(d.Functions, handler)
	patt, err := regexp2.Compile(pattern, 0)
	if err != nil {
		return -1, core.Wrap(lcproc.PatternError, err, "Regexp2.Compile error")
	}
	d.patterns = append(d.patterns, patt)
	if len(d.Functions) != len(d.patterns) {
		return -1, core.Err(lcproc.PatternError, "Can't add pattern: dispatching arrays length is different")
	}
	return len(d.patterns) - 1, nil
}

func (d *Dispatch) getTmp() int {
	d.nameCounter += 1
	return d.nameCounter
}
