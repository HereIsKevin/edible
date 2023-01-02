package parser

import "github.com/HereIsKevin/edible/internal/logger"

type Expr interface {
	Span() logger.Span
}

// String

type ExprStr struct {
	Value     string
	ValueSpan logger.Span
}

func (str *ExprStr) Span() logger.Span {
	return str.ValueSpan
}

// Boolean

type ExprBool struct {
	Value     bool
	ValueSpan logger.Span
}

func (bool *ExprBool) Span() logger.Span {
	return bool.ValueSpan
}

// Integer

type ExprInt struct {
	Value     int64
	ValueSpan logger.Span
}

func (int *ExprInt) Span() logger.Span {
	return int.ValueSpan
}

// Float

type ExprFloat struct {
	Value     float64
	ValueSpan logger.Span
}

func (float *ExprFloat) Span() logger.Span {
	return float.ValueSpan
}

// Reference

type RefModifier uint8

const (
	RefAbsolute RefModifier = iota
	RefRelative
)

type ExprRef struct {
	Modifier     RefModifier
	ModifierSpan logger.Span
	Keys         []Expr
}

func (ref *ExprRef) Span() logger.Span {
	span := ref.ModifierSpan

	if len(ref.Keys) > 0 {
		span.End = ref.Keys[len(ref.Keys)-1].Span().End
	}

	return span
}

// Unary

type UnaryOp uint8

const (
	UnaryPlus UnaryOp = iota
	UnaryMinus
)

type ExprUnary struct {
	Op     UnaryOp
	OpSpan logger.Span
	Right  Expr
}

func (unary *ExprUnary) Span() logger.Span {
	return logger.Span{
		Start: unary.OpSpan.Start,
		End:   unary.Right.Span().End,
	}
}

// Binary

type BinaryOp uint8

const (
	BinaryPlus BinaryOp = iota
	BinaryMinus
	BinaryStar
	BinarySlash
)

type ExprBinary struct {
	Left   Expr
	Op     BinaryOp
	OpSpan logger.Span
	Right  Expr
}

func (binary *ExprBinary) Span() logger.Span {
	return logger.Span{
		Start: binary.Left.Span().Start,
		End:   binary.Right.Span().End,
	}
}

// Array

type ExprArray struct {
	OpenSpan  logger.Span
	Items     []Expr
	CloseSpan logger.Span
}

func (array *ExprArray) Span() logger.Span {
	return logger.Span{
		Start: array.OpenSpan.Start,
		End:   array.CloseSpan.End,
	}
}

// Table

type TableItem struct {
	Key      Expr
	Inherits Expr
	Value    Expr
}

type ExprTable struct {
	OpenSpan  logger.Span
	Items     []TableItem
	CloseSpan logger.Span
}

func (table *ExprTable) Span() logger.Span {
	return logger.Span{
		Start: table.OpenSpan.Start,
		End:   table.CloseSpan.End,
	}
}
