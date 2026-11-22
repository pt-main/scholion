-- ===========================
-- MARK: HELPERS 
-- ===========================
local TYPE_MAP = {
    int = "i", integer = "i", i = "i", i64 = "i",
    float = "f", f = "f", f64 = "f", double = "f",
    string = "s", str = "s", s = "s", text = "s",
    bool = "l", boolean = "l", l = "l", flag = "l",
    bytes = "b", b = "b", base64 = "b",
}

function tess_type(t)
    return TYPE_MAP[t] or t
end

function tess_escape(v)
    if v == nil then return '""' end
    v = tostring(v)
    if v == "" or v:find("[%s\"']") then
        return '"' .. v:gsub("\\", "\\\\"):gsub('"', '\\"') .. '"'
    end
    return v
end

function tess_key(k)
    k = tostring(k)
    if k:find("%s") then
        error("tessera: key must not contain spaces: '" .. k .. "'")
    end
    return k
end

function tess_header(name)
    sys.emitln("<" .. tess_key(name))
end

function tess_footer()
    sys.emitln(">")
end

function tess_entry(key, t, value)
    sys.emitln(string.format("%s %s %s", tess_key(key), tess_type(t), tess_escape(value)))
end

function tess_block(name, fn)
    tess_header(name)
    fn()
    tess_footer()
end


-- ===========================
-- MARK: DIRECTIVES 
-- Arr, Range, Split
-- ===========================

-- lua/schol/arrays.lua

-- //# arr ports i 8000,8001,8002
sys.register([[^\s*arr\s+(?<name>\w+)\s+(?<type>\w+)\s+(?<list>.*)$]], function(args)
    tess_block(args.name, function()
        local idx = 0
        for v in args.list:gmatch("[^,%s]+") do
            tess_entry(idx, args.type, v)
            idx = idx + 1
        end
    end)
end)

-- //# range ports i 8000 8005 2
sys.register([[^\s*range\s+(?<name>\w+)\s+(?<type>\w+)\s+(?<from>-?\d+(?:\.\d+)?)\s+(?<to>-?\d+(?:\.\d+)?)(?:\s+(?<step>-?\d+(?:\.\d+)?))?$]], function(args)
    local step = tonumber(args.step or "1")
    if step == 0 then error("range: step must not be zero") end
    local from, to = tonumber(args.from), tonumber(args.to)
    tess_block(args.name, function()
        local idx, v = 0, from
        while (step > 0 and v <= to) or (step < 0 and v >= to) do
            tess_entry(idx, args.type, tostring(v))
            v = v + step
            idx = idx + 1
        end
    end)
end)

-- //# split tags , s alpha,beta,gamma
sys.register([[^\s*split\s+(?<name>\w+)\s+(?<sep>\S)\s+(?<type>\w+)\s+(?<text>.*)$]], function(args)
    local esc = args.sep:gsub("(%W)", "%%%1")
    tess_block(args.name, function()
        local idx = 0
        for v in args.text:gmatch("[^" .. esc .. "]+") do
            tess_entry(idx, args.type, v)
            idx = idx + 1
        end
    end)
end)

-- ===========================
-- MARK: DIRECTIVES 
-- Enum, Enumm
-- ===========================

-- lua/schol/enums.lua

-- //# enum Color Red Green Blue
sys.register([[^\s*enum\s+(?<name>\w+)\s+(?<values>.+)$]], function(args)
    tess_block(args.name, function()
        local idx = 0
        for v in args.values:gmatch("%S+") do
            tess_entry(v, "i", tostring(idx))
            idx = idx + 1
        end
    end)
end)

-- //# enumm HttpStatus OK=200 NotFound=404
sys.register([[^\s*enumm\s+(?<name>\w+)\s+(?<pairs>.+)$]], function(args)
    tess_block(args.name, function()
        for pair in args.pairs:gmatch("%S+") do
            local k, v = pair:match("([^=]+)=(-?%d+)")
            if k then tess_entry(k, "i", v) end
        end
    end)
end)

-- ===========================
-- MARK: DIRECTIVES 
-- Emitif, Tpl
-- ===========================

-- //# emitif debug true var Debug = true
sys.register([[^\s*emitif\s+(?<name>\w+)\s+(?<val>\S+)\s+(?<out>.*)$]], function(args)
    if flags:get(args.name) == args.val then
        sys.emitln(args.out)
    end
end)

-- //# tpl const VERSION = "{{version}}"
sys.register([[^\s*tpl\s+(?<text>.*)$]], function(args)
    local s = args.text:gsub("{{([%w_]+)}}", function(k)
        return flags:get(k) or ""
    end)
    sys.emitln(s)
end)