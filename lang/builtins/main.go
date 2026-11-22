package builtins

import (
	"fmt"
	"github.com/pt-main/scholion/lang/lcproc"
	luaapi "github.com/pt-main/scholion/lang/luaApi"
	"github.com/pt-main/scholion/lang/processing"
	"github.com/pt-main/scholion/lang/utils"
	"slices"
	"strings"

	"github.com/pt-main/lc/v2/engine/core"
)

var (
	IncludeRegex = `(?s)^\s*include\s+(?<value>.*)`
	EndRegex     = `^\s*end\s*$`
	EchoRegex    = `^\s*echo\s(?<value>.*)`
	EcholnRegex  = `^\s*echoln\s(?<value>.*)`
)

type builtins = map[string]func(*luaapi.LuaApi) processing.DirectiveFunc

var Builtins builtins = builtins{
	IncludeRegex: func(la *luaapi.LuaApi) processing.DirectiveFunc {
		return func(p *processing.Processor, m map[string]string) core.ErrorInterface {
			for _, line := range strings.Split(m["value"], "\n") {
				line = strings.TrimSpace(line)
				spl := strings.Fields(line)
				splen := len(spl)
				if splen == 0 {
					continue
				} else if !slices.Contains([]int{1, 2}, splen) {
					return core.Err(lcproc.IncludeError, `invalid include arglen: %v (must be 1 or 2 - [name] and [luastate])`, splen)
				}
				file := spl[0]
				state := luaapi.DEFAULT_STATE
				if len(spl) > 1 {
					state = spl[1]
				}
				source, err := utils.ReadFileToString(file)
				if err != nil {
					return core.Wrap(lcproc.IncludeError, err, "Can't include file '%v'", file)
				}
				if _, ok := la.States[state]; !ok {
					la.NewLState(state)
				}
				err_ := la.Eval(source, state)
				if err_ != nil {
					return core.Wrap(lcproc.IncludeError, err_, "incluing '%v'", file)
				}

			}
			return nil
		}
	},
	EndRegex: func(la *luaapi.LuaApi) processing.DirectiveFunc {
		return func(p *processing.Processor, m map[string]string) core.ErrorInterface {
			la.Close()
			return core.ErrExit
		}
	},
	EchoRegex: func(la *luaapi.LuaApi) processing.DirectiveFunc {
		return func(p *processing.Processor, m map[string]string) core.ErrorInterface {
			fmt.Print(strings.ReplaceAll(m["value"], "\\n", "\n"))
			return nil
		}
	},
	EcholnRegex: func(la *luaapi.LuaApi) processing.DirectiveFunc {
		return func(p *processing.Processor, m map[string]string) core.ErrorInterface {
			fmt.Println(strings.ReplaceAll(m["value"], "\\n", "\n"))
			return nil
		}
	},
}

func Register(p *processing.Processor, la *luaapi.LuaApi) core.ErrorInterface {
	for re, fn := range Builtins {
		if _, err := p.Disp.AddPattern(re, fn(la)); err != nil {
			return err
		}
	}
	return nil
}
