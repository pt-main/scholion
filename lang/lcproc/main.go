package lcproc

import (
	"fmt"
	"regexp"

	"github.com/pt-main/lc/v2/parsing/stringParsing"
)

type Syntax struct {
	Lexer *stringParsing.Lexer
	Comment,
	CommentBlockStart,
	CommentBlockEnd string
}

func NewSyntax(commentStart, commentBlockStart, commentBlockEnd string) (*Syntax, error) {
	var rules []stringParsing.LexerRule
	var config *stringParsing.LexerConfig = &stringParsing.LexerConfig{}

	rules = append(rules,
		stringParsing.LexerRule{
			Type:    "RAW",
			Pattern: `"(?:[^"\\\n]|\\.)*"`,
		},
		stringParsing.LexerRule{
			Type:    "RAW",
			Pattern: `'(?:[^'\\\n]|\\.)*'`,
		},
		stringParsing.LexerRule{
			Type:    "RAW",
			Pattern: "(?s)`[^`]*`",
		},
	)

	cs := regexp.QuoteMeta(commentStart)

	if commentBlockStart != "" && commentBlockEnd != "" {
		cbs := regexp.QuoteMeta(commentBlockStart)
		cbe := regexp.QuoteMeta(commentBlockEnd)
		rules = append(rules, stringParsing.LexerRule{
			Type:    "DIRECTIVE",
			Pattern: fmt.Sprintf(`(?s)^\s*%s#\s*(?<value>.*?)\s*%s`, cbs, cbe),
		})
	}
	if commentStart != "" {
		rules = append(rules, stringParsing.LexerRule{
			Type:    "DIRECTIVE",
			Pattern: fmt.Sprintf(`(?m)(?<space>\s*)%s#\s*(?<value>[^\n]*)$`, cs),
		})
	}

	rules = append(rules,
		stringParsing.LexerRule{
			Type:    "RAW",
			Pattern: `^.+`,
		},
		stringParsing.LexerRule{
			Type:    "RAW",
			Pattern: `(?s).`,
		},
	)

	l, err := stringParsing.NewLexer(rules, config)
	return &Syntax{
		Lexer:             l,
		Comment:           commentStart,
		CommentBlockStart: commentBlockStart,
		CommentBlockEnd:   commentBlockEnd,
	}, err
}
