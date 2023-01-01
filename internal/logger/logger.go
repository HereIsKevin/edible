package logger

type Span struct {
	Start int
	End   int
}

type Error struct {
	Message string
	Span    Span
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
		return false
	}

	// TODO: Print out and discard errors.

	return true
}

func (logger *Logger) Add(message string, span Span) {
	logger.errors = append(logger.errors, Error{
		Message: message,
		Span:    span,
	})
}
