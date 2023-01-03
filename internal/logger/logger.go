package logger

import "log"

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

type LoggerState struct {
	index int
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

	for _, err := range logger.errors {
		// TODO: Prettier logging.
		log.Println(err)
	}

	// Clear errors.
	logger.errors = nil

	return true
}

func (logger *Logger) Add(message string, span Span) {
	logger.errors = append(logger.errors, Error{
		Message: message,
		Span:    span,
	})
}

func (logger *Logger) Save() LoggerState {
	return LoggerState{index: len(logger.errors)}
}

func (logger *Logger) Restore(state LoggerState) {
	if len(logger.errors) > state.index {
		logger.errors = logger.errors[:state.index]
	}
}
