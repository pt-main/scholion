package luaapi

import (
	"regexp"
	"github.com/pt-main/scholion/lang/lcproc"
	"github.com/pt-main/scholion/lang/processing"
	"strings"

	"github.com/pt-main/lc/v2/engine/core"
	lua "github.com/yuin/gopher-lua"
)

var Regex = `(?s)^\s*lua\s+(?<value>.+)\s*$`

func NewLuaApi(p *processing.Processor, flags map[string]string) *LuaApi {
	la := &LuaApi{
		States: make(map[string]*lua.LState),
		p:      p,
		Flags:  flags,
	}
	la.NewLState(DEFAULT_STATE)
	return la
}

var split = regexp.MustCompile(`\s`)

func RegisterLuaApi(p *processing.Processor, flags map[string]string) (*LuaApi, int, core.ErrorInterface) {
	la := NewLuaApi(p, flags)
	patt, err := p.Disp.AddPattern(Regex, func(p *processing.Processor, m map[string]string) core.ErrorInterface {
		val, ok := m["value"]
		if !ok {
			panic("INVALID LUA API REGEX: value group is not found")
		}
		spl := split.Split(val, 2)
		if len(spl) == 0 {
			return core.Err(lcproc.LuaDirectiveError, "Invalid lua directive arglen: %v (must be >= 1)", len(spl))
		}
		cmd := strings.TrimSpace(spl[0])
		rest := ""
		if len(spl) > 1 {
			rest += strings.TrimSpace(spl[1])
		} else {
			spl := strings.SplitN(val, "\n", 2)
			if len(spl) > 1 {
				rest += strings.TrimSpace(spl[1])
			}
		}

		// newRest := []string{}
		// for _, line := range strings.Split(rest, "\n") {
		// 	newRest = append(newRest, strings.TrimSpace(line))
		// }
		// rest = strings.Join(newRest, "\n")

		// fmt.Println("[c]:", cmd)
		// fmt.Println("[r]:", rest)

		if cmd == "eval" {
			return la.Eval(rest, "")
		} else if strings.HasPrefix(cmd, "eval:") {
			spl := strings.SplitN(cmd, ":", 2)
			state := spl[1]
			return la.Eval(rest, state)
		} else if strings.HasPrefix(cmd, "exec:") {
			count := strings.Count(cmd, ":")
			switch count {
			case 1:
				spl := strings.SplitN(cmd, ":", 2)
				buffer := spl[1]
				return la.ExecBuffer(buffer, "")
			case 2:
				spl := strings.SplitN(cmd, ":", 3)
				buffer := spl[1]
				state := spl[2]
				return la.ExecBuffer(buffer, state)
			}
			return core.Err(lcproc.LuaDirectiveError,
				"invalid exec meta length: %v (must be 2 or 3)", count)
		} else if cmd == "close" {
			if err := la.CloseState(rest); err != nil {
				return err
			}
			return nil
		} else if cmd == "new" {
			la.NewLState(rest)
			return nil
		}
		return core.Err(lcproc.LuaDirectiveError,
			"invalid lua command")
	})
	return la, patt, err
}
