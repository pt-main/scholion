package luaapi

import (
	"fmt"
	"github.com/pt-main/scholion/lang/lcproc"
	"github.com/pt-main/scholion/lang/processing"

	"github.com/pt-main/lc/v2/engine/core"
	lua "github.com/yuin/gopher-lua"
)

// =========================================================
// Main API Functions
// =========================================================

// sys.emit(str1, str2, ...) -> writes strings to the main code buffer
func (la *LuaApi) lua_sys_emit(l *lua.LState) int {
	top := l.GetTop()
	for i := 1; i <= top; i++ {
		la.p.Code = append(la.p.Code, l.CheckString(i))
	}
	return 0
}

// sys.emit(str1, str2, ...) -> writes strings to the main code buffer
func (la *LuaApi) lua_sys_emitln(l *lua.LState) int {
	top := l.GetTop()
	for i := 1; i <= top; i++ {
		la.p.Code = append(la.p.Code, l.CheckString(i))
	}
	la.p.Code = append(la.p.Code, "\n")
	return 0
}

// sys.skip(n) -> skips the next N raw tokens (lines)
func (la *LuaApi) lua_sys_skip(l *lua.LState) int {
	n := l.CheckInt(1)
	la.p.Sys.SkipRawTokens(uint(n))
	return 0
}

// sys.schol(state, code)
// execute schol directive
func (la *LuaApi) lua_sys_schol(l *lua.LState) int {
	if err := la.p.Execute(l.CheckString(1)); err != nil {
		l.RaiseError("schol call error: " + err.Error())
		return 0
	}
	return 0
}

// sys.register(regex, function)
func (la *LuaApi) lua_sys_register(l *lua.LState) int {
	regex := l.CheckString(1)
	fn := l.CheckFunction(2)

	cbId := fmt.Sprintf("cb_%v", la.regCount)
	la.regCount++
	cbTbl := l.GetGlobal("__scholion_callbacks")
	if cbTbl == lua.LNil {
		cbTbl = l.NewTable()
		l.SetGlobal("__scholion_callbacks", cbTbl)
	}
	l.SetField(cbTbl.(*lua.LTable), cbId, fn)

	_, err := la.p.Disp.AddPattern(regex, func(p *processing.Processor, m map[string]string) core.ErrorInterface {
		f := l.GetField(l.GetGlobal("__scholion_callbacks"), cbId)
		l.Push(f)

		tbl := l.NewTable()
		for k, v := range m {
			tbl.RawSetString(k, lua.LString(v))
		}
		l.Push(tbl)

		if err := l.PCall(1, 0, nil); err != nil {
			return core.Wrap(lcproc.LuaError, err, "lua registered callback")
		}
		return nil
	})
	if err != nil {
		l.RaiseError(err.Error())
		return 0
	}

	return 0
}

// =========================================================
// State API Functions
// =========================================================

// state.new(name) -> creates a new isolated lua state
func (la *LuaApi) lua_state_new(l *lua.LState) int {
	name := l.CheckString(1)
	la.NewLState(name)
	return 0
}

// state.close(name) -> destroys a lua state
func (la *LuaApi) lua_state_close(l *lua.LState) int {
	name := l.CheckString(1)
	if err := la.CloseState(name); err != nil {
		l.RaiseError(err.Error())
		return 0
	}
	return 0
}

// state.execute(state, code) -> wraps code in a function, calls it, and returns the result
// Example usage: emit(state.execute("main", "return 1 + 1"))
func (la *LuaApi) lua_state_execute(l *lua.LState) int {
	stateName := l.CheckString(1)
	code := l.CheckString(2)

	L, err := la.GetState(stateName)
	if err != nil {
		l.RaiseError(err.Error())
		return 0
	}

	wrapped := fmt.Sprintf("function _%v() "+code+" end; return _%v()", la.executeCount, la.executeCount)
	la.executeCount += 1

	fn, err_ := L.LoadString(wrapped)
	if err_ != nil {
		l.RaiseError("execute load error: " + err_.Error())
		return 0
	}

	L.Push(fn)

	if err := L.PCall(0, 1, nil); err != nil {
		l.RaiseError("execute call error: " + err.Error())
		return 0
	}

	ret := L.Get(-1)
	L.Pop(1)

	l.Push(ret)
	return 1
}

// =========================================================
// Buffers API Functions
// =========================================================

// buffers.new() -> name (string)
func (la *LuaApi) lua_buffers_new(l *lua.LState) int {
	name := la.p.Sys.NewBuffer()
	l.Push(lua.LString(name))
	return 1
}

// buffers.append(name, str) -> bool
func (la *LuaApi) lua_buffers_append(l *lua.LState) int {
	name := l.CheckString(1)
	str := l.CheckString(2)
	ok := la.p.Sys.AppendToBuffer(str, name)
	l.Push(lua.LBool(ok))
	return 1
}

// buffers.prepend(name, str) -> bool
func (la *LuaApi) lua_buffers_prepend(l *lua.LState) int {
	name := l.CheckString(1)
	str := l.CheckString(2)
	ok := la.p.Sys.PrependToBuffer(str, name)
	l.Push(lua.LBool(ok))
	return 1
}

// buffers.get(name) -> string or nil
func (la *LuaApi) lua_buffers_get(l *lua.LState) int {
	name := l.CheckString(1)
	res, ok := la.p.Sys.GetBuffer(name)
	if !ok {
		l.Push(lua.LNil)
		return 1
	}
	l.Push(lua.LString(res))
	return 1
}

// buffers.clear(name) -> bool
func (la *LuaApi) lua_buffers_clear(l *lua.LState) int {
	name := l.CheckString(1)
	ok := la.p.Sys.ClearBuffer(name)
	l.Push(lua.LBool(ok))
	return 1
}

// buffers.delete(name) -> bool
func (la *LuaApi) lua_buffers_delete(l *lua.LState) int {
	name := l.CheckString(1)
	ok := la.p.Sys.DeleteBuffer(name)
	l.Push(lua.LBool(ok))
	return 1
}

// buffers.paste(name) -> bool (writes buffer content to main code)
func (la *LuaApi) lua_buffers_paste(l *lua.LState) int {
	name := l.CheckString(1)
	ok := la.p.Sys.PasteBuffer(name, la.p)
	l.Push(lua.LBool(ok))
	return 1
}
