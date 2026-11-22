package luaapi

import (
	"fmt"
	"github.com/pt-main/scholion/lang/lcproc"
	"github.com/pt-main/scholion/lang/processing"
	"slices"
	"strings"

	"github.com/pt-main/lc/v2/engine/core"
	lua "github.com/yuin/gopher-lua"
)

const DEFAULT_STATE = "main"

type LuaApi struct {
	States                 map[string]*lua.LState
	p                      *processing.Processor
	Shared                 *lua.LTable
	executeCount, regCount uint64
	Verbose                bool
	Flags                  map[string]string
}

func (la *LuaApi) Close() {
	for _, state := range la.States {
		state.Close()
	}
}

func (la *LuaApi) Eval(code string, state string) core.ErrorInterface {
	if state == "" {
		state = DEFAULT_STATE
	}
	s, err := la.GetState(state)
	if err != nil {
		return err
	}
	err_ := s.DoString(code)
	if err_ != nil {
		return core.Wrap(lcproc.LuaError, err_, "eval on state '%v'", state)
	}
	return nil
}

func (la *LuaApi) ExecBuffer(buffer, state string) core.ErrorInterface {
	if state == "" {
		state = DEFAULT_STATE
	}

	code, ok := la.p.Sys.GetBuffer(buffer)
	if !ok {
		return core.Err(lcproc.LuaError, "exec: buffer '%v' is not found", buffer)
	}

	st, ok := la.States[state]

	if !ok {
		return core.Err(lcproc.LuaError, "exec: state '%v' is not found", state)
	}

	err := st.DoString(code)
	if err != nil {
		return core.Wrap(lcproc.LuaError, err, "exec buffer '%v' on state '%v'", buffer, state)
	}
	return nil
}

func (la *LuaApi) GetState(name string) (*lua.LState, core.ErrorInterface) {
	state, ok := la.States[name]
	if !ok {
		return nil, core.Err(lcproc.LuaError,
			"can't get lua state '%v'", name)
	}
	return state, nil
}

func (la *LuaApi) CloseState(name string) core.ErrorInterface {
	state, err := la.GetState(name)
	if err != nil {
		return err
	}
	state.Close()
	delete(la.States, name)
	return nil
}

func (la *LuaApi) verbose(values ...any) {
	if la.Verbose {
		fmt.Printf("[VERBOSE] ")
		fmt.Println(values...)
	}
}

func (la *LuaApi) NewLState(name string) *lua.LState {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})

	state_flags := []string{}
	if v, has := la.Flags["__state_flags"]; has && len(v) > 0 {
		state_flags = strings.Fields(v)
	}
	if slices.Contains(state_flags, "verbose") {
		la.Verbose = true
	}

	la.verbose("new state :", name, "|", "state flags for", name, ":", state_flags)

	if slices.Contains(state_flags, "open-libs") {
		la.verbose("new state :", name, "|", "opening libs...")
		L.OpenLibs()
		la.verbose("new state :", name, "|", "    ok")
	}

	if slices.Contains(state_flags, "safe-mode") {
		state_flags = append(state_flags, []string{
			"safe-mode:os", "safe-mode:io",
		}...)
	}

	if slices.Contains(state_flags, "safe-mode:os") {
		la.verbose("new state :", name, "|", "safe mode : os...")
		if osLib := L.GetGlobal("os"); osLib != lua.LNil {
			if osTbl, ok := osLib.(*lua.LTable); ok {
				osTbl.RawSetString("execute", lua.LNil)
				osTbl.RawSetString("remove", lua.LNil)
				osTbl.RawSetString("rename", lua.LNil)
				osTbl.RawSetString("exit", lua.LNil)
				osTbl.RawSetString("setlocale", lua.LNil)
				la.verbose("new state :", name, "|", "    ok")
			}
			la.verbose("new state :", name, "|", "end...")
		}
	}
	if slices.Contains(state_flags, "safe-mode:io") {
		la.verbose("new state :", name, "|", "safe mode : io")
		if ioLib := L.GetGlobal("io"); ioLib != lua.LNil {
			if ioTbl, ok := ioLib.(*lua.LTable); ok {
				ioTbl.RawSetString("popen", lua.LNil)
				ioTbl.RawSetString("open", lua.LNil)
				la.verbose("new state :", name, "|", "    ok")
			}
			la.verbose("new state :", name, "|", "end...")
		}
	}

	// ---- Source Api ----
	source := L.NewTable()
	source.RawSetString("__scholion_source", lua.LTrue)
	source.RawSetString("code", L.NewFunction(la.lua_source_code))
	source.RawSetString("line", L.NewFunction(la.lua_source_line))
	source.RawSetString("token", L.NewFunction(la.lua_source_token))

	sourceMeta := L.NewTable()
	sourceMeta.RawSetString("__call", L.NewFunction(la.lua_source_code))
	sourceMeta.RawSetString("__tostring", L.NewFunction(la.lua_source_tostring))
	sourceMeta.RawSetString("__len", L.NewFunction(la.lua_source_len))
	sourceMeta.RawSetString("__concat", L.NewFunction(la.lua_source_concat))
	L.SetMetatable(source, sourceMeta)

	L.SetGlobal("source", source)

	// ---- State Api ----
	la.verbose("new state :", name, "|", "registering state api...")
	state := L.NewTypeMetatable("state")
	L.SetGlobal("state", state)

	L.SetField(state, "execute", L.NewFunction(la.lua_state_execute))
	L.SetField(state, "new", L.NewFunction(la.lua_state_new))
	L.SetField(state, "close", L.NewFunction(la.lua_state_close))
	la.verbose("new state :", name, "|", "    ok")

	// ---- Sys Api ----
	la.verbose("new state :", name, "|", "registering sys api...")
	sys := L.NewTypeMetatable("sys")
	L.SetGlobal("sys", sys)

	L.SetField(sys, "emitln", L.NewFunction(la.lua_sys_emitln))
	L.SetField(sys, "emit", L.NewFunction(la.lua_sys_emit))
	L.SetField(sys, "skip", L.NewFunction(la.lua_sys_skip))
	L.SetField(sys, "schol", L.NewFunction(la.lua_sys_schol))
	L.SetField(sys, "register", L.NewFunction(la.lua_sys_register))
	la.verbose("new state :", name, "|", "    ok")

	// ---- Buffers Api ----
	la.verbose("new state :", name, "|", "registering buf api...")
	buffers := L.NewTypeMetatable("buffers")
	L.SetGlobal("buffers", buffers)

	L.SetField(buffers, "new", L.NewFunction(la.lua_buffers_new))
	L.SetField(buffers, "append", L.NewFunction(la.lua_buffers_append))
	L.SetField(buffers, "prepend", L.NewFunction(la.lua_buffers_prepend))
	L.SetField(buffers, "get", L.NewFunction(la.lua_buffers_get))
	L.SetField(buffers, "clear", L.NewFunction(la.lua_buffers_clear))
	L.SetField(buffers, "delete", L.NewFunction(la.lua_buffers_delete))
	L.SetField(buffers, "paste", L.NewFunction(la.lua_buffers_paste))
	la.verbose("new state :", name, "|", "    ok")

	// ---- Flags ----
	la.verbose("new state :", name, "|", "registering flags api...")
	if la.Flags == nil {
		la.Flags = make(map[string]string)
	}

	helpers := L.NewTable()
	helpers.RawSetString("get", L.NewFunction(la.lua_flags_get))
	helpers.RawSetString("set", L.NewFunction(la.lua_flags_set))
	helpers.RawSetString("delete", L.NewFunction(la.lua_flags_delete))
	helpers.RawSetString("clear", L.NewFunction(la.lua_flags_clear))
	helpers.RawSetString("list", L.NewFunction(la.lua_flags_list))
	helpers.RawSetString("has", L.NewFunction(la.lua_flags_has))

	meta := L.NewTable()
	meta.RawSetString("__helpers", helpers)
	meta.RawSetString("__index", L.NewFunction(la.lua_flags_index))
	meta.RawSetString("__newindex", L.NewFunction(la.lua_flags_newindex))
	meta.RawSetString("__pairs", L.NewFunction(la.lua_flags_pairs))
	meta.RawSetString("__len", L.NewFunction(la.lua_flags_len))

	flags := L.NewTable()
	flags.Metatable = meta
	for k, v := range la.Flags {
		flags.RawSetString(k, lua.LString(v))
	}
	L.SetGlobal("flags", flags)
	la.verbose("new state :", name, "|", "    ok")

	la.verbose("new state :", name, "|", "adding scope...")
	if la.Shared == nil {
		la.Shared = L.NewTable()
	}
	L.SetGlobal("scope", la.Shared)
	la.verbose("new state :", name, "|", "    ok")

	la.States[name] = L

	la.verbose("new state :", name, "|", "compleate")
	return L
}
