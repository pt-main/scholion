// server.go.schol
package main

import "fmt"

//# lua eval
//# lua eval os.execute("echo starting...")

//# lua eval sys.emitln("\nimport \"net/http\"")

//# include test.lua main

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
	x := "//# not a directive"
	fmt.Println("Server starting...")
	//# echoln finish...
}
