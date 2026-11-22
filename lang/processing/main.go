package processing

import (
	"github.com/pt-main/scholion/lang/lcproc"
	"strings"

	"github.com/pt-main/lc/v2/engine"
	"github.com/pt-main/lc/v2/engine/core"
	"github.com/pt-main/lc/v2/parsing/stringParsing"
)

func (p *Processor) RawHandler(e engine.StringEngineInterface, pn *stringParsing.ParsedNode) core.ErrorInterface {
	if p.Sys.skipRawTokens > 0 {
		p.Sys.skipRawTokens -= 1
		return nil
	}
	p.Sys.LinePos += uint(len(strings.Split(pn.Raw, "\n")) - 1)
	p.Code = append(p.Code, pn.Raw)
	p.Sys.TokenPos += 1
	return nil
}

func (p *Processor) Execute(s string) core.ErrorInterface {
	pattName := -1
	groups := make(map[string]string)

	for name, patt := range p.Disp.patterns {
		m, err := patt.FindStringMatchStartingAt(s, 0)
		if err != nil {
			return core.Wrap(lcproc.WrappedError, err, "matching %v", name)
		}
		if m == nil || m.Index != 0 {
			continue
		}

		pattName = name
		// fmt.Println(pattName, patt.String(), s)

		for _, g := range m.Groups() {
			if g.Name == "0" || len(g.Captures) == 0 {
				continue
			}
			groups[g.Name] = g.String()
		}
		break
	}

	if pattName == -1 {
		return core.Err(lcproc.MatchError, "No matched patterns")
	}

	err := p.Disp.Functions[pattName](p, groups)
	if err != nil {
		return core.Wrap(lcproc.WrappedError, err, "executing %v", pattName)
	}

	return nil
}

func (p *Processor) DirectiveHandler(e engine.StringEngineInterface, pn *stringParsing.ParsedNode) core.ErrorInterface {
	p.Sys.TokenPos += 1
	v, _ := pn.Metadata["value"].(string)
	p.Sys.LinePos += uint(len(strings.Split(v, "\n")) - 1)
	// s, _ := pn.Metadata["space"].(string)
	// p.Code = append(p.Code, s)
	// fmt.Println("DIR:", v)
	return p.Execute(p.EscapeDirective(v))
}
