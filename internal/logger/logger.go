package logger

import "fmt"

type Pos struct {
	Start int
	End   int
	Line  int
}

type Error struct {
	Message string
	Pos     Pos
}

func (err *Error) Error() string {
	return err.Message
}

type Logger struct {
	source string
	errors []Error
}

func New(source string) *Logger {
	return &Logger{
		source: source,
		errors: []Error{},
	}
}

func (logger *Logger) Log() bool {
	if len(logger.errors) == 0 {
		// Did not have errors.
		return false
	}

	// Log every error.
	for _, err := range logger.errors {
		fmt.Printf(
			"[Line %d] Error at '%s': %s\n",
			err.Pos.Line,
			logger.source[err.Pos.Start:err.Pos.End],
			err.Message,
		)
	}

	// Clear errors.
	logger.errors = nil

	// Had errors.
	return true
}

func (logger *Logger) Add(message string, pos Pos) {
	logger.errors = append(logger.errors, Error{
		Message: message,
		Pos:     pos,
	})
}

func (logger *Logger) AddError(err error) {
	logger.errors = append(logger.errors, *err.(*Error))
}
