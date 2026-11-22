package processing

import (
	"github.com/pt-main/scholion/lang/lcproc"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/pt-main/lc/v2/engine/core"
)

type System struct {
	Buffers       map[string][]string
	TokenPos      uint
	LinePos       uint
	bufCount      uint64
	skipRawTokens uint
}

type DirectiveFunc = func(*Processor, map[string]string) core.ErrorInterface // map - regexp2 group values

type Dispatch struct {
	patterns    []*regexp2.Regexp
	sources     []string
	nameCounter int
	Functions   []DirectiveFunc
}

type Processor struct {
	Code   []string
	Disp   *Dispatch
	Sys    *System
	Syntax *lcproc.Syntax
}

func (p *Processor) GetCode() string {
	return strings.Join(p.Code, "")
}

func NewSystem() *System {
	return &System{
		Buffers: make(map[string][]string),
	}
}

func NewDispatch() *Dispatch {
	return &Dispatch{
		patterns:  make([]*regexp2.Regexp, 0),
		Functions: make([]DirectiveFunc, 0),
	}
}

func NewProcessor() *Processor {
	return &Processor{
		Code:   make([]string, 0),
		Sys:    NewSystem(),
		Disp:   NewDispatch(),
		Syntax: nil,
	}
}
