// Package diag renders scholion errors for humans.
//
// The model and layout follow tycl's diag package: every problem becomes a
// flat Diagnostic (code, severity, message, hint, span), and the renderer
// prints it with the source line and a caret, causes indented below:
//
//	error String:PROCESS_ERR2: source processing failed
//	 hint: processing stopped at the first failing directive; the cause chain explains why
//
//	 error DEFAULT_EVENTS:CONTEXT_ERROR: failed while processing the highlighted line
//	  input.go:3:1
//	  hint: this is the source line being processed when the error occurred
//	     3 | //# lua eval print(nil.x)
//	       | ^^^^^^^^^^^^^^^^^^^^^^^^^
//
//	  error ScholionFromLuaError: eval on state 'main'
//	   hint: lua raised while running eval/exec; the innermost cause holds the lua message
//
//	  error <string>:1: attempt to index a nil value
//
//	1 error in input.go
//
// Scholion errors are lc errors (core.Err / core.Wrap chains), so Collect
// walks the ErrorInterface chain instead of building diagnostics from
// scratch. The lc string engine attaches the failing raw line to
// DefaultEventsCallErrorContexted errors under the EMK(0, "string") meta
// key; when the source text is passed in, that line is located in it and
// highlighted. Plumbing levels that only relay their cause (plain
// EVENT_ERROR, "Call loop failed", "Iteration failed") are dropped, and the
// first occurrence of the line is used, so a repeated identical line may be
// highlighted at its earlier position.
//
// Colour codes ([?RD], [?YW], ...) are tap color placeholders. They are
// resolved once, at the very end, so color.ColorEnabled = false and
// redirected output need no separate code path.
//
// Usage:
//
//	err := lang.Process(source)
//	if err != nil {
//	        fmt.Println(diag.RenderIn(err, source, inputPath))
//	}
package utils

import (
	goerr "errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/pt-main/lc/v2/engine/core"
	lcerrors "github.com/pt-main/lc/v2/public/errors"
	"github.com/pt-main/tap/go/color"

	"github.com/pt-main/scholion/lang/lcproc"
)

// Tap color placeholders, resolved once by color.Set at the end of Render.
const (
	accent = "[?YW]"
	strong = "[?BBK]"
	bad    = "[?RD]"
	good   = "[?GN]"
	info   = "[?BE]"
	reset  = "[?RT]"
)

// maxDepth caps the cause-chain walk, so a cyclic Unwrap cannot hang a
// render. Real scholion chains are a dozen levels deep at most.
const maxDepth = 64

// rawLineMeta is the key under which the lc string engine stores the raw
// line being processed when a handler fails (core.EMK(0, "string")).
var rawLineMeta = core.EMK(0, "string")

// Severity classifies how much a diagnostic matters.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Span is a 1-based inclusive source range.
type Span struct {
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
	Width     int    `json:"width,omitempty"`
}

// String renders a span as file:line:column.
func (s *Span) String() string {
	if s == nil {
		return ""
	}
	switch {
	case s.File != "" && s.Line > 0:
		return fmt.Sprintf("%s:%d:%d", s.File, s.Line, s.Column)
	case s.File != "":
		return s.File
	case s.Line > 0:
		return fmt.Sprintf("%d:%d", s.Line, s.Column)
	default:
		return ""
	}
}

// Excerpt renders the referenced line with the span range split out, so the
// caller can colour only the offending part. A span wider than the line is
// clamped to the line end, which is what happens with block directives.
func (s *Span) Excerpt(text string) (before, marked, after string, ok bool) {
	if s == nil || s.Line < 1 {
		return "", "", "", false
	}
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if s.Line > len(lines) {
		return "", "", "", false
	}
	runes := []rune(lines[s.Line-1])
	col := s.Column - 1
	if col < 0 {
		col = 0
	}
	if col > len(runes) {
		col = len(runes)
	}
	width := s.Width
	if width < 1 {
		width = 1
	}
	end := col + width
	if end > len(runes) {
		end = len(runes)
	}
	if end <= col {
		end = col
		if end > len(runes) {
			end = len(runes)
		}
	}
	return string(runes[:col]), string(runes[col:end]), string(runes[end:]), true
}

// CaretLine builds the underline for the span, aligned under Excerpt.
func (s *Span) CaretLine(before, marked string) string {
	caret := strings.Repeat(" ", len([]rune(before)))
	if marked == "" {
		return caret + "^"
	}
	return caret + strings.Repeat("^", len([]rune(marked)))
}

// Diagnostic is one flat, self-describing problem report. A cause chain
// becomes a tree of these; rendering keeps the report order, so (unlike
// tycl's validation diagnostics) they are never sorted or deduped.
type Diagnostic struct {
	Code     string            `json:"code"`
	Severity Severity          `json:"severity"`
	Message  string            `json:"message"`
	Hint     string            `json:"hint,omitempty"`
	Span     *Span             `json:"span,omitempty"`
	Label    string            `json:"label,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
	Causes   []Diagnostic      `json:"causes,omitempty"`
}

// Summary renders a one-line description without position info.
func (d Diagnostic) Summary() string {
	msg := d.Code
	if d.Message != "" {
		msg = d.Message
	}
	if d.Span != nil {
		if loc := d.Span.String(); loc != "" {
			return fmt.Sprintf("%s: %s", loc, msg)
		}
	}
	return msg
}

// HasErrors reports whether the list contains at least one error.
func HasErrors(list []Diagnostic) bool {
	for _, d := range list {
		if d.Severity != SeverityWarning {
			return true
		}
	}
	return false
}

// hints maps known scholion and lc error codes to a "what to do about it"
// line, mirroring tycl's contract hints.
var hints = map[lcerrors.ErrorCodeType]string{
	// scholion
	lcproc.WrappedError:           "a directive handler returned an error; the reason is in the cause chain below",
	lcproc.MatchError:             "no registered pattern matches the directive text; check its syntax, or register a handler from lua with sys.register(regex, fn)",
	lcproc.PatternError:           "the pattern is invalid or already registered; patterns use regexp2 syntax (github.com/dlclark/regexp2)",
	lcproc.LuaDirectiveError:      "valid forms: 'lua eval <code>', 'lua eval:<state> <code>', 'lua exec:<buffer>', 'lua exec:<buffer>:<state>', 'lua new <state>', 'lua close <state>'",
	lcproc.LuaError:               "lua raised while running eval/exec; the innermost cause holds the lua message",
	lcproc.DirectiveFunctionError: "a directive function rejected its input; see the cause chain below",

	// lc engine
	lcerrors.StringEngineProcessError1:       "the lexer could not tokenize the source; check the comment syntax (comment, commentBlockStart, commentBlockEnd)",
	lcerrors.StringEngineProcessError2:       "processing stopped at the first failing directive; the cause chain explains why",
	lcerrors.DefaultEventsCallErrorContexted: "this is the source line being processed when the error occurred",
	lcerrors.DefaultEventsCallErrorHandler:   "the engine command handler for this node failed; the reason is in the cause chain below",
	lcerrors.DefaultEventsPanicError:         "a handler panicked; this is usually a bug in scholion itself or in a registered lua callback",
	lcerrors.DefaultEventsCallErrorUnknown:   "the lexer produced a token type with no registered engine command (expected RAW or DIRECTIVE)",
	lcerrors.ParsingError:                    "the source could not be tokenized with the configured comment syntax",
}

// HintFor returns the hint for a known error code, or "".
func HintFor(code string) string {
	return hints[lcerrors.ErrorCodeType(code)]
}

// Collect turns an error into a tree of diagnostics: lc errors become one
// diagnostic per chain level (nested as causes), plain Go errors become
// code-less leaves. When source is given, the failing raw line stored by
// the lc engine is located in it and becomes the highlighted span.
func Collect(err error, source string) []Diagnostic {
	if err == nil {
		return nil
	}
	if root := collect(err, source, 0); root != nil {
		return []Diagnostic{*root}
	}
	return nil
}

// isNilError reports whether err is nil, including a typed nil pointer
// stored in the interface (a nil *core.Error would panic the getters).
func isNilError(err error) bool {
	if err == nil {
		return true
	}
	switch v := reflect.ValueOf(err); v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return v.IsNil()
	default:
		return false
	}
}

func collect(err error, source string, depth int) *Diagnostic {
	// Unwrap chains end in a nil interface, and a typed nil (a nil *core.Error
	// stored in an error) would panic the getters below; stop on both.
	if isNilError(err) {
		return nil
	}
	if depth > maxDepth {
		return &Diagnostic{
			Code:     "DiagChainTooDeep",
			Severity: SeverityError,
			Message:  fmt.Sprintf("cause chain deeper than %d levels; truncated", maxDepth),
		}
	}

	d := Diagnostic{Severity: SeverityError}

	if ei, ok := err.(core.ErrorInterface); ok {
		d.Code = ei.GetCode()
		d.Message = ei.GetMsg()
		// Process errors embed the whole chain pre-rendered in their
		// message; the causes below already carry it, so shorten it.
		switch lcerrors.ErrorCodeType(d.Code) {
		case lcerrors.StringEngineProcessError1:
			d.Message = "source parsing failed"
		case lcerrors.StringEngineProcessError2:
			d.Message = "source processing failed"
		}
		if meta := ei.GetMeta(); len(meta) > 0 {
			d.Meta = make(map[string]string, len(meta))
			for k, v := range meta {
				d.Meta[string(k)] = fmt.Sprintf("%v", v)
			}
		}
		d.Hint = hints[lcerrors.ErrorCodeType(d.Code)]
		span := spanFor(ei, source)
		if span != nil {
			d.Span = span
			// The default message embeds the whole raw line quoted, which
			// would duplicate the excerpt below; replace it in that case.
			if lcerrors.ErrorCodeType(d.Code) == lcerrors.DefaultEventsCallErrorContexted {
				d.Message = "failed while processing the highlighted line"
			}
		}
		// Plumbing levels ("Event handler failed", "Call loop failed",
		// "Iteration failed") only relay their cause; drop them so the
		// tree keeps the informative reports only.
		if isPlumbing(ei) {
			return collect(ei.Unwrap(), source, depth+1)
		}
		d.Causes = collectChildren(ei.Unwrap(), source, depth+1)
		return &d
	}

	// Plain Go error: no code, no meta; keep unwrapping the standard way.
	d.Message = strings.TrimRight(err.Error(), " \t\r\n")
	d.Causes = collectChildren(goerr.Unwrap(err), source, depth+1)
	return &d
}

func collectChildren(cause error, source string, depth int) []Diagnostic {
	if child := collect(cause, source, depth); child != nil {
		return []Diagnostic{*child}
	}
	return nil
}

// isPlumbing reports whether an lc error level carries no information of
// its own: an EVENT_ERROR that only relays its cause, or a context error
// without the raw-line meta (the ones with the meta mark the failing line
// and are kept).
func isPlumbing(ei core.ErrorInterface) bool {
	code := lcerrors.ErrorCodeType(ei.GetCode())
	if isNilError(ei.Unwrap()) {
		return false
	}
	switch code {
	case lcerrors.EventsEventError:
		return true
	case lcerrors.DefaultEventsCallErrorContexted:
		return ei.GetMeta()[rawLineMeta] == nil
	default:
		return false
	}
}

// spanFor derives a span from the raw line the lc engine attaches to
// context errors. It needs the source text and returns nil when the line
// cannot be located in it.
func spanFor(ei core.ErrorInterface, source string) *Span {
	if source == "" {
		return nil
	}
	raw, ok := ei.GetMeta()[rawLineMeta].(string)
	if !ok {
		return nil
	}
	return locateSpan(source, raw)
}

// locateSpan finds the first occurrence of raw in source and converts it
// to a 1-based line/column span, counted in runes.
//
// Directive nodes carry the newline before them in their Raw, because the
// directive lexer rule matches across the line boundary; that newline is
// an artifact, so it is trimmed and the span points at the directive line.
func locateSpan(source, raw string) *Span {
	raw = strings.TrimLeft(raw, "\r\n")
	if raw == "" || raw == "[NIL]" {
		return nil
	}
	idx := strings.Index(source, raw)
	if idx < 0 {
		return nil
	}
	head := source[:idx]
	line := 1 + strings.Count(head, "\n")
	col := len([]rune(head)) + 1
	if lastNL := strings.LastIndex(head, "\n"); lastNL >= 0 {
		col = len([]rune(head[lastNL+1:])) + 1
	}
	return &Span{Line: line, Column: col, Width: len([]rune(raw))}
}

// Render formats an error against its source text, with the summary line.
// The result is fully colour-resolved and has no trailing newline.
func Render(err error, source string) string {
	return RenderIn(err, source, "")
}

// RenderIn is Render with a file name for the derived spans, so locations
// read like "input.go:3:1" instead of "3:1".
func RenderIn(err error, source, file string) string {
	list := Collect(err, source)
	if file != "" {
		for i := range list {
			walkSpans(&list[i], func(s *Span) { s.File = file })
		}
	}
	return RenderList(list, source)
}

// RenderList formats diagnostics for a terminal, with source excerpts and
// carets, and appends the summary line.
func RenderList(list []Diagnostic, text string) string {
	if len(list) == 0 {
		return ""
	}
	parts := make([]string, 0, len(list))
	for _, d := range list {
		parts = append(parts, renderOne(d, text, ""))
	}
	out := strings.TrimRight(strings.Join(parts, "\n"), "\n")
	if s := Summary(list); s != "" {
		out += "\n\n" + s
	}
	return color.Set(out)
}

func walkSpans(d *Diagnostic, f func(*Span)) {
	if d.Span != nil {
		f(d.Span)
	}
	for i := range d.Causes {
		walkSpans(&d.Causes[i], f)
	}
}

func renderOne(d Diagnostic, text string, indent string) string {
	var b strings.Builder
	head := d.Message
	if d.Label != "" && !strings.Contains(head, d.Label) {
		head = strings.TrimSpace(d.Label + ": " + head)
	}
	if d.Code != "" {
		head = d.Code + ": " + head
	}
	tag := "error"
	if d.Severity == SeverityWarning {
		tag = "warning"
	}
	// Plain error text (lua messages, tracebacks) may span lines: keep the
	// continuation lines aligned under the head.
	head = strings.ReplaceAll(head, "\n", "\n"+indent+"    ")

	fmt.Fprintf(&b, "%s%s%s %s%s\n", indent, bad, tag, reset, head)
	if loc := location(d); loc != "" {
		fmt.Fprintf(&b, "%s %s%s%s\n", indent, info, loc, reset)
	}
	if d.Hint != "" {
		fmt.Fprintf(&b, "%s %shint:%s %s\n", indent, accent, reset, d.Hint)
	}
	if d.Span != nil && d.Span.Line > 0 {
		if before, marked, after, ok := d.Span.Excerpt(text); ok {
			num := fmt.Sprintf("%4d", d.Span.Line)
			gutter := info + strings.Repeat(" ", len(num)) + reset + " | "
			fmt.Fprintf(&b, "%s %s%s%s | %s%s%s%s\n",
				indent, info, num, reset, before, bad, marked, after)
			fmt.Fprintf(&b, "%s %s%s%s%s\n",
				indent, gutter, strong, d.Span.CaretLine(before, marked), reset)
		}
	}
	for _, c := range d.Causes {
		b.WriteString("\n")
		b.WriteString(renderOne(c, text, indent+"  "))
	}
	return b.String()
}

func location(d Diagnostic) string {
	if d.Span == nil {
		return ""
	}
	return d.Span.String()
}

// Summary renders a count line such as "2 errors in input.go". A cause
// chain is one problem, so only the top-level diagnostics are counted;
// span files are collected through the whole tree.
func Summary(list []Diagnostic) string {
	if len(list) == 0 {
		return ""
	}
	errs, warns := 0, 0
	files := map[string]bool{}
	var gather func(d Diagnostic)
	gather = func(d Diagnostic) {
		if d.Span != nil && d.Span.File != "" {
			files[d.Span.File] = true
		}
		for _, c := range d.Causes {
			gather(c)
		}
	}
	for _, d := range list {
		if d.Severity == SeverityWarning {
			warns++
		} else {
			errs++
		}
		gather(d)
	}
	parts := []string{}
	if errs > 0 {
		parts = append(parts, pluralize(errs, "error"))
	}
	if warns > 0 {
		parts = append(parts, pluralize(warns, "warning"))
	}
	out := strings.Join(parts, ", ")
	switch len(files) {
	case 1:
		for name := range files {
			out += " in " + name
		}
	case 0:
	default:
		out += fmt.Sprintf(" in %d files", len(files))
	}
	return out
}

// Success renders a confirmation line.
func Success(text string) string {
	return color.Set(good + text + reset)
}

// Failure renders an error summary line.
func Failure(text string) string {
	return color.Set(bad + text + reset)
}

// pluralize renders "1 error" / "3 errors".
func pluralize(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
