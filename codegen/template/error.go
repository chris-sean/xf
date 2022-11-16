package template

const ERROR_GO = `package errs

import "github.com/chris-sean/xf"

func ConflictError(err any) xf.ErrorType {
	return xf.NewErrorType("ConflictError", 409, err)
}
`
