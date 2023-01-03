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

type Logger struct {
	source string
	errors []Error
	saves  []int
}

func New(source string) *Logger {
	return &Logger{
		source: source,
		errors: []Error{},
		saves:  []int{},
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

func (logger *Logger) Save() {
	logger.saves = append(logger.saves, len(logger.errors))
}

func (logger *Logger) Restore() {
	logger.errors = logger.errors[:logger.saves[len(logger.saves)-1]]
	logger.saves = logger.saves[:len(logger.saves)-1]
}
