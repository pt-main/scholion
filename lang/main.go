package lang

import (
	"context"
	"github.com/pt-main/scholion/lang/builtins"
	"github.com/pt-main/scholion/lang/lcproc"
	luaapi "github.com/pt-main/scholion/lang/luaApi"
	"github.com/pt-main/scholion/lang/processing"

	"github.com/pt-main/lc/v2"
	"github.com/pt-main/lc/v2/engine"
	"github.com/pt-main/lc/v2/engine/core"
	"github.com/pt-main/lc/v2/public"
)

func NewEngine(syntax *lcproc.Syntax, ctx context.Context, p *processing.Processor) *engine.StringEngine {
	e := lc.NewStringEngine(
		public.StringResType, []string{"main"},
		true, syntax.Lexer, ctx,
	)
	p.Syntax = syntax
	e.UEP.Scope[public.StringEngineScopeCanBeUnknown] = false
	// l := e.UEP.Logger.(*core.Logger)
	// l.Logging[public.LogEvents] = true
	// l.Logging[public.LogParsing] = true
	// l.Logging[public.LogVerbose] = true
	e.NewCommandFull("RAW", p.RawHandler, "Add raw code to main buffer")
	e.NewCommandFull("DIRECTIVE", p.DirectiveHandler, "Parse shcolion directive")
	return e
}

func NewLang(syntax *lcproc.Syntax, flags map[string]string) (*engine.StringEngine, *processing.Processor, core.ErrorInterface) {
	p := processing.NewProcessor()
	la, _, err := luaapi.RegisterLuaApi(p, flags)
	if err != nil {
		return nil, p, err
	}
	err = builtins.Register(p, la)
	if err != nil {
		return nil, p, err
	}
	e := NewEngine(syntax, context.Background(), p)
	return e, p, nil
}
