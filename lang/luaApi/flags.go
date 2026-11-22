package luaapi

import lua "github.com/yuin/gopher-lua"

func (la *LuaApi) lua_flags_index(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)

	if mt, ok := t.Metatable.(*lua.LTable); ok {
		if helpers, ok := mt.RawGetString("__helpers").(*lua.LTable); ok {
			if v := helpers.RawGetString(key); v != lua.LNil {
				L.Push(v)
				return 1
			}
		}
	}

	L.Push(lua.LNil)
	return 1
}

func (la *LuaApi) lua_flags_newindex(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)
	v := L.Get(3)
	if v == lua.LNil {
		t.RawSetString(key, lua.LNil)
	} else {
		t.RawSetString(key, lua.LString(v.String()))
	}
	return 0
}

func (la *LuaApi) lua_flags_pairs(L *lua.LState) int {
	t := L.CheckTable(1)
	L.Push(L.GetGlobal("next"))
	L.Push(t)
	L.Push(lua.LNil)
	return 3
}

func (la *LuaApi) lua_flags_len(L *lua.LState) int {
	t := L.CheckTable(1)
	n := 0
	t.ForEach(func(_, _ lua.LValue) { n++ })
	L.Push(lua.LNumber(n))
	return 1
}

func (la *LuaApi) lua_flags_get(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)
	L.Push(t.RawGetString(key))
	return 1
}

func (la *LuaApi) lua_flags_set(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)
	v := L.Get(3)
	if v == lua.LNil {
		t.RawSetString(key, lua.LNil)
	} else {
		t.RawSetString(key, lua.LString(v.String()))
	}
	return 0
}

func (la *LuaApi) lua_flags_delete(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)
	t.RawSetString(key, lua.LNil)
	return 0
}

func (la *LuaApi) lua_flags_clear(L *lua.LState) int {
	t := L.CheckTable(1)
	var keys []lua.LValue
	t.ForEach(func(k, _ lua.LValue) { keys = append(keys, k) })
	for _, k := range keys {
		t.RawSet(k, lua.LNil)
	}
	return 0
}

func (la *LuaApi) lua_flags_list(L *lua.LState) int {
	t := L.CheckTable(1)
	res := L.NewTable()
	t.ForEach(func(k, v lua.LValue) {
		res.RawSet(k, v)
	})
	L.Push(res)
	return 1
}

func (la *LuaApi) lua_flags_has(L *lua.LState) int {
	t := L.CheckTable(1)
	key := L.CheckString(2)
	L.Push(lua.LBool(t.RawGetString(key) != lua.LNil))
	return 1
}
