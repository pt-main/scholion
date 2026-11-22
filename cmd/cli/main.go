package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"github.com/pt-main/scholion/lang"
	"github.com/pt-main/scholion/lang/lcproc"
	"github.com/pt-main/scholion/lang/utils"
	"strings"

	tap "github.com/pt-main/tap/go"
	tessera "github.com/pt-main/tessera/impl/go"
)

var Version = "1.0.0"

func New() *tap.Parser {
	p := tap.NewParser("schol", `[?GN]╭───────
│ [?BD][?YW]scholion[?RT] - Language-agnostic preprocessor with Lua macros.
[?GN]| [?RT]By [?UE]Pt[?RT], [?BD]humanmade[?RT].
[?GN]╰───────[?RT]`, []string{"help", "-h"}, tap.DefaultParserConfig())

	p.AddCommand("version", func(p *tap.Parser, s []string) error {
		fmt.Printf("Schol lang v%v\n", Version)
		return nil
	}, "Show version", nil, nil, false)

	p.AddCommand(tap.DEFAULT_CMD, func(p *tap.Parser, s []string) error {
		input := s[0]
		out := ""
		if len(s) > 1 {
			out = s[1]
		} else {
			ext := filepath.Ext(input)
			if ext == ".schol" {
				ext = filepath.Ext(strings.TrimRight(input, ".schol"))
			}
			out = filepath.Base(input) + "_compiled" + ext
		}
		source, err := utils.ReadFileToString(input)
		if err != nil {
			return err
		}

		var (
			c   = "//"
			cbs = "/*"
			cbe = "*/"
		)
		values := p.Flags
		flags := map[string]string{}
		if conf, has := p.Flags["config"]; has {
			source, err := utils.ReadFileToString(conf)
			if err != nil {
				return err
			}
			conf, err_ := tessera.Parse(source)
			if err_ != nil {
				errors_ := []string{}
				for _, err := range err_ {
					errors_ = append(errors_, err.Error())
				}
				errors := "    |" + strings.Join(errors_, "\n    |\n    |")
				return fmt.Errorf("Parsing tessera: \n%v", errors)
			}
			values = conf.Strs
			if flags_, has := conf.Inners["flags"]; has {
				flags = flags_.Strs
			}
			for k, v := range p.Flags {
				values[k] = v
			}
		}
		if c_, has := values["comment"]; has {
			c = strings.TrimSpace(c_)
		}
		if cbs_, has := values["commentBlockStart"]; has {
			cbs = strings.TrimSpace(cbs_)
		}
		if cbe_, has := values["commentBlockEnd"]; has {
			cbe = strings.TrimSpace(cbe_)
		}

		synt, err := lcproc.NewSyntax(c, cbs, cbe)
		if err != nil {
			return err
		}
		lang, proc, err := lang.NewLang(synt, flags)
		if err != nil {
			return err
		}
		err = lang.Process(source)
		if err != nil {
			return errors.New(utils.RenderIn(err, source, input))
		}
		if err := utils.WriteStringToFile(out, proc.GetCode()); err != nil {
			return err
		}
		return nil
	}, `
[?YW]Usage:[?RT]
  schol [--config=<file>] <input> [output]

[?YW]Arguments:[?RT]
  [?GN]input[?RT]   Path to the source file to process.
  [?GN]output[?RT]  Path to the destination file. Defaults to <input>_compiled.<ext>.

[?YW]Flags:[?RT]
  [?GN]--config=<file>[?RT]    Path to a Tessera configuration file.
  [?GN]--[comment|commentBlockStart|commentBlockEnd]=<...>[?RT]
    Syntax configuration from cli (if has no config)

[?YW]Config File Format (Tessera):[?RT]
  comment           string  Line comment prefix (default: "//")
  commentBlockStart string  Block comment start (default: "/*")
  commentBlockEnd   string  Block comment end   (default: "*/")

  <flags
    # ignoring not-string keys...
    key string "value"
  >
  The <flags> inner table is passed to the Lua environment as the global 'flags' table.

  Tessera: https://github.com/pt-main/tessera

[?YW]Directives:[?RT]
  Directives must start at the beginning of a line.
  Line directive:   <comment>#<directive>
  Block directive:  <blockStart>#<directive><blockEnd>

  [?CN]Example (for C/Go syntax):[?RT]
  //# lua eval sys.emit("hello")
  /*# lua eval sys.emit("world") */
`, []string{"input"}, []string{"output"}, false)
	return p
}
