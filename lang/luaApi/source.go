package luaapi

import lua "github.com/yuin/gopher-lua"

// =========================================================
// source: access to code generated so far + current position
// =========================================================

// source() and source.code() -> current code as string
func (la *LuaApi) lua_source_code(l *lua.LState) int {
	l.Push(lua.LString(la.p.GetCode()))
	return 1
}

// source.line() -> 1-based current source line
func (la *LuaApi) lua_source_line(l *lua.LState) int {
	l.Push(lua.LNumber(la.p.Sys.LinePos + 1))
	return 1
}

// source.token() -> 1-based current token
func (la *LuaApi) lua_source_token(l *lua.LState) int {
	l.Push(lua.LNumber(la.p.Sys.TokenPos + 1))
	return 1
}

// __tostring -> current code
func (la *LuaApi) lua_source_tostring(l *lua.LState) int {
	l.Push(lua.LString(la.p.GetCode()))
	return 1
}

// __len -> byte length of current code
func (la *LuaApi) lua_source_len(l *lua.LState) int {
	l.Push(lua.LNumber(len(la.p.GetCode())))
	return 1
}

// __concat -> source .. "x" and "x" .. source
func (la *LuaApi) lua_source_concat(l *lua.LState) int {
	a := l.Get(1)
	b := l.Get(2)
	code := la.p.GetCode()
	l.Push(lua.LString(sourceOperand(a, code) + sourceOperand(b, code)))
	return 1
}

func sourceOperand(v lua.LValue, code string) string {
	if t, ok := v.(*lua.LTable); ok {
		if t.RawGetString("__scholion_source") == lua.LTrue {
			return code
		}
	}
	return v.String()
}
