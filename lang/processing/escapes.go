package processing

import "strings"

func (p *Processor) EscapeDirective(s string) string {
	runes := []rune(s)
	var res strings.Builder
	res.Grow(len(s))

	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			switch runes[i+1] {
			case '\\':
				res.WriteRune('\\')
				i++
				continue
			case 'C':
				res.WriteString(p.Syntax.Comment)
				i++
				continue
			// case 'S':
			// 	res.WriteString(p.Syntax.CommentBlockStart)
			// 	i++
			// 	continue
			case 'E':
				res.WriteString(p.Syntax.CommentBlockEnd)
				i++
				continue
			}
		}
		res.WriteRune(runes[i])
	}
	return res.String()
}
