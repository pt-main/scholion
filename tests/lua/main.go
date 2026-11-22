package main

import (
	"context"
	"fmt"
	"github.com/pt-main/scholion/lang"
	"github.com/pt-main/scholion/lang/lcproc"
	luaapi "github.com/pt-main/scholion/lang/luaApi"
	"github.com/pt-main/scholion/lang/processing"
)

func main() {
	l, err := lcproc.NewSyntax("//", "/*", "*/")
	fmt.Println(err)
	p := processing.NewProcessor()
	luaapi.RegisterLuaApi(p, nil)
	e := lang.NewEngine(l, context.Background(), p)
	fmt.Println(e.Process(`
//# lua eval print("start...")
/*# lua eval 
local name = buffers.new()
print("name:", name)
buffers.append(name, [=[
	scope.x = scope.x + 1
	print(scope.x)
	print(state.execute("tmp", "print(x)\nreturn 'test result'"))
	sys.register('\\s*#.*', function (args) end)
	sys.emit("\S \n    multyline comment!!\n\E\n")
]=])

*/
//# lua eval sys.schol("lua new tmp")
//# lua eval scope.x = 10
//# lua eval:tmp x = "hello from tmp!"
//# lua exec:__tmp_buf_n0
//# lua close tmp
//# lua eval print(scope.x)
//# # skips \n (after last non-from-start-of-line pattern) and 'skipped':
//# lua eval sys.skip(2)
//# lua eval sys.emit("test emmited line")
skipped
pasted`))
	fmt.Println(p.GetCode())
}
