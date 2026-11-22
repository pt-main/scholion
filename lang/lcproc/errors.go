package lcproc

import "github.com/pt-main/lc/v2/public/errors"

const (
	WrappedError           errors.ErrorCodeType = "WrappedScholionError"
	MatchError             errors.ErrorCodeType = "ScholionMatchError"
	PatternError           errors.ErrorCodeType = "ScholionInvalidPatternError"
	LuaDirectiveError      errors.ErrorCodeType = "ScholionFromLuaDirectiveError"
	LuaError               errors.ErrorCodeType = "ScholionFromLuaError"
	IncludeError           errors.ErrorCodeType = "ScholionFromIncludeDirectiveError"
	DirectiveFunctionError errors.ErrorCodeType = "ScholionDirectiveFunctionError"
)
