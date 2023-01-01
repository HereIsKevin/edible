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
