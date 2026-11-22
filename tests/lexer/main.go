package main

import (
	"fmt"
	"github.com/pt-main/scholion/lang/lcproc"
)

func main() {
	syn, err := lcproc.NewSyntax("//", "/*", "*/")
	fmt.Println(err)
	nodes, err := syn.Lexer.Parse(`
// server.go.schol
package main

import "fmt"

//# lua eval sys.emitln("\nimport \"net/http\"")

//# lua eval print("starting...")

//# lua eval for i=1, 3 do sys.emit(string.format("\n// Generated var %d", i)) end
//# lua eval for i=1, 3 do sys.emit(string.format("\nvar Port%d int = %d", i, 8000+i)) end

// DSL will be: ROUTE: <method> <path> => <handler>
/*# lua eval
sys.emitln()
sys.register([[^\s*ROUTE:\s*(?<method>\w+)\s+(?<path>\\S+)\s*=>\s*(?<handler>\w+)]], function(args)
    sys.emitln(string.format("// Auto-generated handler: %s %s", args.method, args.path))
    sys.emitln(string.format("func %s(w http.ResponseWriter, r *http.Request) {", args.handler))
    sys.emitln("    w.WriteHeader(http.StatusOK)")
    sys.emitln('    fmt.Fprintf(w, "Handling %s %s", r.Method, r.URL.Path)')
    sys.emitln("}")
    sys.emitln("")
end)
*/

/*# lua eval
sys.emitln()
if flags:has("debug") and flags:get("debug") == "true" then
    sys.emitln('var DebugMode bool = true')
    sys.emitln('// Debug mode is ON!')
else
    sys.emitln('var DebugMode bool = false')
end
*/

//# ROUTE: GET /users => HandleGetUsers
//# ROUTE: POST /users => HandleCreateUser

func main() {
	//# lua eval sys.skip(2)
	x := "//# not a derictive"
	fmt.Println("Server starting...")
	//# lua eval print("finish...")
}
`)
	fmt.Println(err)
	for _, node := range nodes {
		fmt.Printf("[%s] %s\n\n\n", node.Switch, node.Raw)
	}
}
