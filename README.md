# Scholion

> **Language-agnostic preprocessor with Lua macros.**

[![Go](https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Made with lc](https://img.shields.io/badge/engine-pt--main%2Flc-purple)](https://github.com/pt-main/lc)

Scholion turns your source file into a program that generates itself. Directives embedded in comments are executed by an embedded Lua interpreter, which can generate code, manage buffers, register custom DSLs, and share data between isolated states. The output is a ready-to-compile file.

---

## Table of Contents

1. [The Problem](#the-problem)
2. [The Solution](#the-solution)
3. [Why This Way](#why-this-way)
4. [Installation](#installation)
5. [Quick Start](#quick-start)
6. [Directives](#directives)
7. [Lua API](#lua-api)
8. [Custom DSL via `sys.register`](#custom-dsl-via-sysregister)
9. [Configuration (Tessera)](#configuration-tessera)
10. [CLI](#cli)
11. [Examples](#examples)
12. [Internals](#internals)
13. [Project Structure](#project-structure)
14. [Errors & Debugging](#errors--debugging)
15. [Limitations & Trade-offs](#limitations--trade-offs)
16. [License](#license)

---

## The Problem

### 1. Code generation lives separately from code

Today, metaprogramming is almost always kept outside the source file:

- `protoc`, `mockgen`, `stringer`, `sqlc`, `go generate` - separate utilities;
- `text/template`, `Jinja`, `mustache` - separate template files;
- `m4`, `Cog`, `scons`-style - separate build passes.

The problem is always the same: **the generated output is separated from what created it**. Changes in the template are not visible at the usage site, context is lost, navigation breaks, and error diagnostics point to a "foreign" file. As a result, you have to keep two worlds in your head and synchronize them manually.

### 2. Many languages lack macros

- **Go** has no macros (generics do not replace them for code generation).
- **C/C++** has only a text preprocessor, which knows nothing about types, loops, or reflection.
- **Java, C#, Python, TypeScript** - either reflection, external code generation, or nothing.

Developers face a dilemma: either write boilerplate code by hand or build an external generator.

### 3. Existing tools are tied to a single language

`go generate` works only with Go. `cargo-expand` only with Rust. `cpp` only with C/C++. If you have a monorepo with five languages or your own DSL, you need **one** tool, not five.

### 4. Specialized preprocessors are too weak

`m4` is powerful, but its syntax is painful. `Cog` uses Python but is tied to Python. `scons` is a builder, not a preprocessor. None of them provide simultaneously:

- embedding directly into the source file,
- a full-fledged scripting language,
- registration of custom patterns,
- independence from the target language.

**Conclusion:** we need a tool that works *inside* the file, on *any* language, provides a *full* scripting language, and allows *extending itself* for the task.

---

## The Solution

### Three Pillars of Scholion

1. **Comments as directive carriers.** The file remains valid for the target language - IDEs, linters, and compilers notice nothing.
2. **Lua as the macro language.** Small (~450 KB), fast, embeddable (`gopher-lua`, pure Go), and familiar to millions of developers.
3. **Custom DSLs via `sys.register`.** You define your own patterns and get a mini-language right inside comments.

### How it looks

Before processing - `server.go.schol`:

```go
package main

import "fmt"

//# lua eval print("starting...")
//# lua eval sys.emitln("\nimport \"net/http\"")

//# lua eval for i=1, 3 do sys.emit(string.format("\nvar Port%d int = %d", i, 8000+i)) end

// DSL: ROUTE: <method> <path> => <handler>
/*# lua eval
sys.register([[^\s*ROUTE:\s*(?<method>\w+)\s+(?<path>\S+)\s*=>\s*(?<handler>\w+)]], function(args)
    sys.emitln(string.format("func %s(w http.ResponseWriter, r *http.Request) {", args.handler))
    sys.emitln("    w.WriteHeader(http.StatusOK)")
    sys.emitln("}")
end)
*/

//# ROUTE: GET /users => HandleGetUsers
//# ROUTE: POST /users => HandleCreateUser

func main() {
	fmt.Println("Server starting...")
}
```

After `schol server.go.schol` - `server.go.schol_compiled.go`:

```go
package main

import "fmt"
import "net/http"

var Port1 int = 8001
var Port2 int = 8002
var Port3 int = 8003

func HandleGetUsers(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}

func main() {
	fmt.Println("Server starting...")
}
```

**The file remains compilable at all stages.** Directives are just comments for the compiler and macros for Scholion.

---

## Why This Way

### Why comments, not special syntax?

| Approach | Pros | Cons |
|---|---|---|
| **Special Syntax** (`#macro`, `@gen`) | Explicit, highlighted | Breaks target language parser, requires forks |
| **Separate Files** (`*.gen.go`) | Clean | Context lost, two sources of truth |
| **Comments** (our choice) | File valid, works everywhere, IDE-friendly | Visual noise if overused |

Comments are the only place in a language where you can **guaranteedly** write anything without breaking grammar. Scholion uses this as an entry point. If you overuse directives, it’s a style issue, not an architectural one.

### Why Lua, not Python / JS / custom language?

- **`gopher-lua`** - pure Go, no cgo, ~450 KB binary size, embeds in one line.
- **Startup speed** - fractions of a millisecond vs. hundreds for CPython.
- **Simple C-API** - easy to control sandboxing and stack.
- **Familiar syntax** - tables, closures, metatables, `string.format`, `gmatch` - everything is already there.
- **Alternatives are worse**: Python pulls in cgo and tens of megabytes; JS engines are huge; a custom language means years of work.

Lua is "powerful enough and cheap enough". If you need a DSL, you build it on top of Lua (`sys.register` + closures) rather than writing a parser from scratch.

### Why `regexp2`, not standard `regexp`?

Standard `regexp` in Go is RE2: linear time, but **no**:

- lookahead/lookbehind,
- backreferences,
- named groups in .NET syntax `(?<name>...)`,
- lazy quantifiers in full volume.

Scholion uses `github.com/dlclark/regexp2` - a port of .NET Regex. This gives:

- **named groups** `(?<value>...)` - the entire API is built on them (`m["value"]`);
- **lazy quantifiers** `.*?` for accurate block capturing;
- **flags** `(?s)`, `(?m)` in one place;
- **backtracking** - predictable behavior for typical DSL patterns.

The price is potential catastrophic backtracking vulnerability on hostile patterns. In Scholion, the user writes the patterns, so this is a conscious trade-off: flexibility is more important than paranoia.

### Why Tessera, not YAML/JSON/TOML?

Tessera is a configuration format tailored for:

- **flat strings** (`comment s //`) without quotes;
- **nested tables** (`<flags ... >`) - exactly what is needed to pass flags to Lua;
- **typing** (`s`, `i`, `f`, `l`), if needed.

YAML would be redundant and dangerous (the Norway problem), JSON has too many quotes for a five-line config, TOML is also overkill. Tessera is "JSON for people who are too lazy to type quotes". The same format is used by `run` and other projects by the author - an ecosystem.

### Why buffers, states, and shared `scope`?

These are three different mechanisms for three different tasks:

- **Buffers** - accumulate output fragments and then insert them entirely (`paste`). Useful when code is generated non-linearly: first collect the "header", then the "body", then glue them.
- **States** - isolation. An included file should not accidentally overwrite global variables of the main script. `lua new tmp` creates a separate Lua world.
- **Shared `scope`** - the single table shared by all states. Data is passed between isolated worlds through it.

Without separation, you’d get either "everything in one global space" (conflicts) or "everything isolated" (impossible to communicate). Three levels are the minimum for real scenarios.

### Why first-match-wins in dispatching?

In `Dispatch.patterns`, patterns are iterated **in registration order**, and the first match wins. This is:

- **predictable** - you control the order by calling `AddPattern`;
- **extensible** - a user can register a more specific pattern *before* the built-in one;
- **debuggable** - if a pattern "ate" someone else's directive, just reorder the registration.

### Why a lexer, not line-by-line `strings.HasPrefix`?

Strings, block comments, and multiline directives are three things that break a naive line-by-line parser:

- `x := "//# not a directive"` - a string, not a directive;
- `/*# ... \n ... \n */` - a block, not a line;
- `` `...` `` - backticks with newlines.

The lexer first extracts **string literals** with separate rules, and only then looks for directives. This is the only way not to confuse data with control structures.

---

## Installation

```bash
go install github.com/pt-main/github.com/pt-main/scholion/cmd/schol@latest
```

---

## Quick Start

### 1. Create a file

`hello.go.schol`:

```go
package main

import "fmt"

//# lua eval print("generating...")
//# lua eval for i=1, 3 do sys.emitln(string.format("var N%d = %d", i, i)) end

func main() {
	fmt.Println("hello")
}
```

### 2. Run it

```bash
schol hello.go.schol
```

### 3. See the result

`hello.go.schol_compiled.go`:

```go
package main

import "fmt"

var N1 = 1
var N2 = 2
var N3 = 3

func main() {
	fmt.Println("hello")
}
```

By default, the output file is `<input>_compiled.<ext>`. Explicitly:

```bash
schol hello.go.schol hello.go
```

---

## Directives

### Syntax

A directive is a comment starting with `#` immediately after the comment marker. Leading whitespace is preserved (indentation is not lost).

**Line** (for `//`):

```
<comment>#<directive>
```

**Block** (for `/* ... */`):

```
<blockStart>#<directive><blockEnd>
```

Block form supports multiline content:

```
/*# lua eval
sys.emitln("line 1")
sys.emitln("line 2")
*/
```

Comment syntax is configurable via CLI (`--comment`, `--commentBlockStart`, `--commentBlockEnd`) or via config (see below). Default is C/Go/PHP style: `//`, `/*`, `*/`.

### What is NOT a directive

- Strings: `x := "//# not a directive"` - `//#` inside a string literal is not parsed.
- Comments not at the start of a line: `foo //# not a directive` - a directive must be at the start of a line (leading whitespace allowed).
- Unknown directives: `//# unknown foo` → error `No matched patterns` (see [Errors](#errors--debugging)).

### Escaping

Escaping is supported inside directive values to generate characters that conflict with comment syntax:

| Sequence | Result |
|---|---|
| `\\` | `\` |
| `\C` | Line comment prefix (`//`) |
| `\E` | Block comment end (`*/`) |

Example:

```
/*# lua eval sys.emit("before\n\E\nafter") */
```

Outputs `before\n*/\nafter` - i.e., a literal `*/` in the generated code, without closing the current block.

### Lua Commands

Inside `lua` directives, the following commands are available:

| Command | Description |
|---|---|
| `lua eval <code>` | Execute code in the default state (`main`). |
| `lua eval:<state> <code>` | Execute code in the specified state. |
| `lua exec:<buffer>` | Execute code from a buffer in `main`. |
| `lua exec:<buffer>:<state>` | Execute code from a buffer in the specified state. |
| `lua new <state>` | Create a new isolated Lua state. |
| `lua close <state>` | Close a state. |

Command parsing: the first word after `lua` (up to space/newline) is the command, the rest is the argument. Thus `lua eval print("x")` and `lua eval\nprint("x")` are equivalent.

### Other Directives

Besides `lua`, Scholion registers four built-in directives:

| Directive | Description |
|---|---|
| `include <file> [state]` | Read a file and execute it as Lua code in state `state` (default `main`). If the state doesn't exist, it will be created. |
| `end` | Close **all** Lua states and finish processing. |
| `echo <text>` | Print text to stdout without a newline (`\n` in text → newline). |
| `echoln <text>` | Print text to stdout with a newline. |

---

## Lua API

Each state has access to global tables `sys`, `state`, `buffers`, `flags`, and the shared `scope`.

### Security and Sandbox
By default, standard Lua libraries are disabled for security reasons. Macros cannot execute system commands or read arbitrary files from disk.

To access Lua stdlib, use the special flag `__state_flags` of type `s`. It contains space-separated flags that define how the Lua state is created. Available flags:

- `open-libs` - enables the standard library.
- `safe-mode:io` - restricts `io.open`, `io.popen`.
- `safe-mode:os` - restricts `os.execute`, `os.remove`, `os.rename`, `os.exit`, `os.setlocale`.
- `safe-mode` - enables all the above restrictions.
- `verbose` - adds debug data to the output.

For example, to enable Lua stdlib with security restrictions:

```text
<flags
  __state_flags s "open-libs safe-mode"
>
```

### `sys` - output and registration

```lua
sys.emit("a", "b", "c")          -- write strings to the main code buffer
sys.emitln("a", "b", "c")        -- same + "\n"
sys.skip(2)                      -- skip the next 2 RAW tokens
sys.schol("lua eval print('x')") -- execute a schol directive recursively
sys.register(pattern, fn)        -- register a DSL pattern
```

`sys.skip(n)` is useful when you generate code instead of original lines: the next `n` lines from the file will not appear in the output.

`sys.schol(code)` - recursive call: the passed string undergoes the same dispatching as a regular directive. Convenient if you build a directive dynamically.

`sys.register(regex, fn)` - see separate [section](#custom-dsl-via-sysregister).

### `state` - isolated states

```lua
state.new("tmp")                       -- create a state
state.execute("tmp", "return 1 + 1")   -- execute and get result
state.close("tmp")                     -- close
```

`state.execute(state, code)` wraps code in a function, calls it, and returns the last `return`:

```lua
local r = state.execute("tmp", "local x = 8; return x + 2")
print(r) -- 10
```

This is the only way to get a value from another state into the current one.

### `buffers` - accumulators

Buffers live at the processor level (`Processor.Sys.Buffers`) and are **shared** between all states.

```lua
local name = buffers.new()           -- "__tmp_buf_n0"
buffers.append(name, "a")            -- -> ["a"]
buffers.prepend(name, "z")           -- -> ["z", "a"]
local s = buffers.get(name)          -- "za"
buffers.paste(name)                  -- write "za" to main code
buffers.clear(name)                  -- clear
buffers.delete(name)                 -- delete
```

Typical scenario: collect several fragments in a buffer, then insert them as one piece - so you don't worry about output order.

### `flags` - flags from config

```lua
flags.debug                 -- direct access
flags:get("debug")          -- via method
flags:set("debug", "true")
flags:has("debug")
flags:delete("debug")
flags:clear()
flags:list()                -- copy of table
```

Flags are initialized from the `<flags>` section in the Tessera config and do **not** mutate the `Processor` - the table lives in a specific state. Changes in one state are not visible in another (except those initially present in the config). This is an intentional limitation; use `scope` for exchange.

### `scope` - shared table

One Lua table for the entire processor, shared by all states:

```lua
-- in main
scope.counter = 0

-- in tmp
scope.counter = scope.counter + 1
print(scope.counter) -- 1
```

This is the recommended way to exchange data between isolated states.

### `source` - current code and position

Global `source` gives access to what has already been accumulated in the main buffer `Processor.Code`, as well as the current position in the source. All values are read dynamically at the time of the call.

```lua
local c = source()            -- current code as string
local c = source.code()       -- same, explicit alias
local ln = source.line()      -- current line (1-based)
local tk = source.token()     -- current token (1-based)

print(#source)                -- length of code in bytes
print(tostring(source))       -- same as source()
print("--- begin ---\n" .. source .. "\n--- end ---")
```

Methods are regular functions, so they can be called via `:` or `.`:

```lua
print(source:line())          -- equivalent to source.line()
```

---

## Custom DSL via `sys.register`

`sys.register(pattern, fn)` adds a pattern to the dispatcher. When a directive string matches `pattern`, `fn` is called with a table of named groups.

```lua
-- Register "enum Name V1 V2 V3"
sys.register([[^\s*enum\s+(?<name>\w+)\s+(?<values>.+)$]], function(args)
    sys.emitln("type " .. args.name .. " int")
    sys.emitln("const (")
    local i = 0
    for v in args.values:gmatch("%S+") do
        sys.emitln(string.format("    %s%s %s = %d",
            args.name, v, args.name, i))
        i = i + 1
    end
    sys.emitln(")")
end)
```

Usage:

```
//# enum Color Red Green Blue
```

Rules:

- Pattern must start with `^\s*` - the directive comes with trimmed `#` and space, but with leading line whitespace.
- Named groups `(?<name>...)` go into `args.name`.
- First matching pattern wins. Registration order = priority order.
- If the pattern fails to compile, `sys.register` raises a Lua error.

Library of ready-made DSLs (`arr`, `range`, `split`, `enum`, `enumm`, `emitif`, `tpl`) is in [`libs/tessera.lua`](libs/tessera.lua). It can be connected via `include`:

```
//# include libs/tessera.lua main
```

---

## Configuration (Tessera)

Example `scholion.tess`:

```
comment s //
commentBlockStart s /*
commentBlockEnd s */

<flags
  debug s "true"
  version s "1.0.0"
>
```

- Top level (`comment`, `commentBlockStart`, `commentBlockEnd`) - comment syntax.
- Section `<flags>` - table accessible in Lua as `flags`.

Run:

```bash
schol --config=scholion.tess input.go.schol output.go
```

CLI flags (`--comment=...`) override values from the config.

---

## CLI

```
schol [--config=<file>] [--comment=<...>] \
      [--commentBlockStart=<...>] [--commentBlockEnd=<...>] \
      <input> [output]
```

| Argument / Flag | Description |
|---|---|
| `input` | Path to the source file. |
| `output` | Path to the output file. Defaults to `<input>_compiled.<ext>`. |
| `--config=<file>` | Tessera config. |
| `--comment=<...>` | Line comment prefix (default `//`). |
| `--commentBlockStart=<...>` | Block comment start (default `/*`). |
| `--commentBlockEnd=<...>` | Block comment end (default `*/`). |

Subcommands:

- `schol version` - version.
- `schol help` / `schol -h` - help.

CLI is built on [`tap`](https://github.com/pt-main/tap).

---

## Examples

### Port Generation

```
//# lua eval for i=1, 3 do sys.emit(string.format("\nvar Port%d int = %d", i, 8000+i)) end
```

### HTTP Routes as DSL

```
/*# lua eval
sys.register([[^\s*ROUTE:\s*(?<method>\w+)\s+(?<path>\S+)\s*=>\s*(?<handler>\w+)]], function(args)
    sys.emitln(string.format("func %s(w http.ResponseWriter, r *http.Request) {", args.handler))
    sys.emitln("    w.WriteHeader(http.StatusOK)")
    sys.emitln("}")
end)
*/

//# ROUTE: GET /users => HandleGetUsers
//# ROUTE: POST /users => HandleCreateUser
```

### Conditional Output by Flags

```
/*# lua eval
if flags:has("debug") and flags:get("debug") == "true" then
    sys.emitln("var DebugMode bool = true")
else
    sys.emitln("var DebugMode bool = false")
end
*/
```

### Buffer + Deferred Insertion

```
/*# lua eval
local h = buffers.new()
buffers.append(h, "package main\n")
buffers.append(h, "import \"fmt\"\n")
buffers.paste(h)
*/
```

### Include Reusable Library

`libs/tessera.lua`:

```lua
sys.register([[^\s*enum\s+(?<name>\w+)\s+(?<values>.+)$]], function(args) ... end)
```

Usage:

```
//# include libs/tessera.lua main
//# enum Color Red Green Blue
```

### Hiding Source Lines

```
//# lua eval sys.skip(2)
this line is a placeholder
this one too
```

The first two text tokens after the directive will not appear in the output.

---

## Internals

### Pipeline

```
          ┌──────────┐    ┌───────────┐    ┌─────────────┐    ┌──────────┐
source -> │  Lexer   │ -> │ Processor │ -> │ Dispatching │ -> │ Lua API  │ -> output
          │ (lc)     │    │ (RAW/DIR) │    │ (regexp2)   │    │ (gopher) │
          └──────────┘    └───────────┘    └─────────────┘    └──────────┘
```

1. **Lexer** (`lcproc.Syntax.Lexer`) cuts the input stream into `RAW` and `DIRECTIVE` tokens. String literals and block comments are recognized by separate rules - so `"//# not a directive"` does not become a directive.
2. **Processor** (`processing.Processor`) processes tokens:
   - `RAW` → `RawHandler` → into `Code` (or skipped if `sys.skip`);
   - `DIRECTIVE` → `DirectiveHandler` → `Execute`.
3. **Dispatching** iterates registered `regexp2` patterns and calls the first match.
4. **Lua API** (`luaApi.LuaApi`) provides `sys`, `state`, `buffers`, `flags`, `scope` and executes Lua code via `gopher-lua`.

### Processor

- `Code []string` - final output (joined via `strings.Join`).
- `Sys.Buffers map[string][]string` - named buffers.
- `Sys.skipRawTokens uint` - counter of skipped RAW tokens.
- `Disp` - pattern dispatcher.

`GetCode()` returns `strings.Join(p.Code, "")`.

### Dispatching

```go
type Dispatch struct {
    patterns    []*regexp2.Regexp
    sources     []string
    Functions   []DirectiveFunc
    nameCounter int
}
```

`AddPattern(source, handler)` compiles `regexp2` and adds to the end. `Execute(s)`:

1. Iterates `patterns` in order.
2. For the first match - collects named groups into `map[string]string`.
3. Calls `Functions[idx](p, groups)`.
4. If nothing matched - `MatchError`.

### Lua API

- `LuaApi.States map[string]*lua.LState` - all states.
- `LuaApi.p *processing.Processor` - reference to processor (for `sys.emit` and `buffers`).
- `LuaApi.Shared *lua.LTable` - `scope` table, shared by all states.
- `LuaApi.Flags map[string]string` - initial flags (used when creating new states).

Each `NewLState` creates a `gopher-lua` state and registers `sys`, `state`, `buffers`, `flags`, `scope` tables in it.

---

# Project Structure

```text
github.com/pt-main/scholion/
├── cmd/
│   ├── schol/
│   │   └── main.go             # CLI entry point
│   └── cli/
│       └── main.go             # Argument parsing via tap, Tessera integration
├── lang/
│   ├── main.go                 # StringEngine initialization and component binding
│   ├── lcproc/
│   │   ├── errors.go           # Scholion error codes
│   │   └── main.go             # Lexer configuration (RAW and DIRECTIVE rules)
│   ├── processing/
│   │   ├── struct.go           # Definition of Processor, Dispatch, System
│   │   ├── main.go             # Token handlers (RawHandler, DirectiveHandler)
│   │   ├── dispatching.go      # AddPattern and Execute logic
│   │   ├── escapes.go          # Escaping \C, \E, \\
│   │   └── systems.go          # Buffer management (New, Append, Paste, Skip)
│   ├── luaApi/
│   │   ├── structs.go          # Lua state initialization
│   │   ├── main.go             # Registration of lua eval/exec/new/close
│   │   ├── functions.go        # Implementation of sys.*, state.*, buffers.*
│   │   ├── flags.go            # Metatable and flags methods
│   │   └── source.go           # Access to generated code and positions
│   ├── builtins/
│   │   └── main.go             # Built-in directives
│   └── utils/
│       ├── main.go             # File read/write utilities
│       └── errfmt.go           # Advanced error rendering with carets and hints
├── libs/
│   └── tessera.lua             # Library of ready-made DSLs (arr, range, enum, tpl)
├── tests/
│   ├── lexer/main.go           # Tokenization tests
│   ├── engine/main.go          # Pipeline tests
│   ├── lua/main.go             # State isolation and buffer tests
│   └── example/                # Full HTTP server generation example
├── go.mod
├── go.sum
├── LICENSE                     # Apache 2.0
└── README-ru.md                # This file
```


---

## Errors & Debugging

Error codes are defined in [`lang/lcproc/errors.go`](lang/lcproc/errors.go):

| Code | When it occurs |
|---|---|
| `WrappedScholionError` | Wrapper for any error inside a handler. |
| `ScholionMatchError` | No pattern matched (`No matched patterns`). |
| `ScholionInvalidPatternError` | `regexp2.Compile` failed to compile the pattern. |
| `ScholionFromLuaDirectiveError` | Error parsing `lua` command (`eval`, `exec`, `new`, `close`). |
| `ScholionFromLuaError` | Lua code execution error. |
| `ScholionFromIncludeDirectiveError` | `include` error (file not found, Lua error). |
| `ScholionDirectiveFunctionError` | Error inside a user-defined handler. |

### Typical Problems

- **`No matched patterns`** - directive not recognized. Check that the pattern is registered (via `lua eval` or `include`), and that registration order didn't let a wider pattern "eat" the directive.
- **Directive inside a string worked** - means the string was parsed as RAW, but you expected otherwise. Check quotes (`"..."`, `'...'`, `` `...` ``).
- **`\E` didn't work** - escaping works only in the directive *value*, not in the Lua code body. In Lua, use `"\\E"`.
- **Flags not visible in another state** - expected: `flags` mutates locally. Use `scope`.

### Debug Points

In `lang/main.go`, lines are commented out to enable detailed `lc` logging:

```go
// l := e.UEP.Logger.(*core.Logger)
// l.Logging[public.LogEvents] = true
// l.Logging[public.LogParsing] = true
// l.Logging[public.LogVerbose] = true
```

Uncomment them to get a stream of parsing events in stdout.

---

## Limitations & Trade-offs

Conscious limitations of the current version:

1. **No AST analysis.** Scholion works with text and comments, does not understand the semantics of the target language. Consequence: works on any language, but cannot, for example, "rename all function calls".
2. **Directive order is critical.** First matching pattern wins; wide patterns can suppress narrow ones.
3. **`flags` are not shared between states.** Only `scope` is shared.
4. **`end` closes all states at once.** No way to close "all except one".
5. **`sys.register` is bound to the registering state.** If the state is closed, its registered patterns remain in the dispatcher, but the callback will fail when called.
6. **Patterns are regex, not parsers.** For complex DSLs with nesting, it is better to build a parser in Lua.

---

## License

Apache 2.0 - see [LICENSE](LICENSE).

---

*By Pt, 2026. Built on [`lc`](https://github.com/pt-main/lc), [`tap`](https://github.com/pt-main/tap), [`tessera`](https://github.com/pt-main/tessera) and [`gopher-lua`](https://github.com/yuin/gopher-lua).*