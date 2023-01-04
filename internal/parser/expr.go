package parser

import (
	"fmt"

	"github.com/HereIsKevin/edible/internal/logger"
)

type Expr interface {
	Span() logger.Span
	fmt.Stringer
}

// String

type ExprStr struct {
	Value     string
	ValueSpan logger.Span
}

func (str *ExprStr) Span() logger.Span {
	return str.ValueSpan
}

func (str *ExprStr) String() string {
	return fmt.Sprintf("Str(\"%s\")", str.Value)
}

// Boolean

type ExprBool struct {
	Value     bool
	ValueSpan logger.Span
}

func (bool *ExprBool) Span() logger.Span {
	return bool.ValueSpan
}

func (bool *ExprBool) String() string {
	return fmt.Sprintf("Bool(%t)", bool.Value)
}

// Integer

type ExprInt struct {
	Value     int64
	ValueSpan logger.Span
}

func (int *ExprInt) Span() logger.Span {
	return int.ValueSpan
}

func (int *ExprInt) String() string {
	return fmt.Sprintf("Int(%d)", int.Value)
}

// Float

type ExprFloat struct {
	Value     float64
	ValueSpan logger.Span
}

func (float *ExprFloat) Span() logger.Span {
	return float.ValueSpan
}

func (float *ExprFloat) String() string {
	return fmt.Sprintf("Float(%f)", float.Value)
}

// Reference

type RefModifier uint8

const (
	RefAbsolute RefModifier = iota
	RefRelative
)

func (modifier RefModifier) String() string {
	switch modifier {
	case RefAbsolute:
		return "Absolute"
	case RefRelative:
		return "Relative"
	default:
		return "Unknown"
	}
}

type ExprRef struct {
	Modifier     RefModifier
	ModifierSpan logger.Span
	Keys         []Expr

	// Used by interpreter.
	Cache Expr
}

func (ref *ExprRef) Span() logger.Span {
	span := ref.ModifierSpan

	if len(ref.Keys) > 0 {
		span.End = ref.Keys[len(ref.Keys)-1].Span().End
	}

	return span
}

func (ref *ExprRef) String() string {
	keys := []string{}

	for _, key := range ref.Keys {
		keys = append(keys, key.String())
	}

	cache := "nil"

	if ref.Cache != nil {
		cache = ref.Cache.String()
	}

	return debugStruct("Ref", []debugField{
		{"Modifier", ref.Modifier.String()},
		{"Keys", debugSlice(keys)},
		{"Cache", cache},
	})
}

// Unary

type UnaryOp uint8

const (
	UnaryPlus UnaryOp = iota
	UnaryMinus
)

func (op UnaryOp) String() string {
	switch op {
	case UnaryPlus:
		return "Plus"
	case UnaryMinus:
		return "Minus"
	default:
		return "Unknown"
	}
}

type ExprUnary struct {
	Op     UnaryOp
	OpSpan logger.Span
	Right  Expr

	// Used by interpreter.
	Cache Expr
}

func (unary *ExprUnary) Span() logger.Span {
	return logger.Span{
		Start: unary.OpSpan.Start,
		End:   unary.Right.Span().End,
	}
}

func (unary *ExprUnary) String() string {
	cache := "nil"

	if unary.Cache != nil {
		cache = unary.Cache.String()
	}

	return debugStruct("Unary", []debugField{
		{"Op", unary.Op.String()},
		{"Right", unary.Right.String()},
		{"Cache", cache},
	})
}

// Binary

type BinaryOp uint8

const (
	BinaryPlus BinaryOp = iota
	BinaryMinus
	BinaryStar
	BinarySlash
)

func (op BinaryOp) String() string {
	switch op {
	case BinaryPlus:
		return "Plus"
	case BinaryMinus:
		return "Minus"
	case BinaryStar:
		return "Star"
	case BinarySlash:
		return "Slash"
	default:
		return "Unknown"
	}
}

type ExprBinary struct {
	Left   Expr
	Op     BinaryOp
	OpSpan logger.Span
	Right  Expr

	// Used by interpreter.
	Cache Expr
}

func (binary *ExprBinary) Span() logger.Span {
	return logger.Span{
		Start: binary.Left.Span().Start,
		End:   binary.Right.Span().End,
	}
}

func (binary *ExprBinary) String() string {
	cache := "nil"

	if binary.Cache != nil {
		cache = binary.Cache.String()
	}

	return debugStruct("Binary", []debugField{
		{"Left", binary.Left.String()},
		{"Op", binary.Op.String()},
		{"Right", binary.Right.String()},
		{"Cache", cache},
	})
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

func (array *ExprArray) String() string {
	items := []string{}

	for _, item := range array.Items {
		items = append(items, item.String())
	}

	return debugStruct("Array", []debugField{
		{"Items", debugSlice(items)},
	})
}

// Table

type TableItem struct {
	Key      Expr
	Inherits Expr
	Value    Expr
}

func (item *TableItem) String() string {
	parent := "nil"

	if item.Inherits != nil {
		parent = item.Inherits.String()
	}

	return debugStruct("", []debugField{
		{"Key", item.Key.String()},
		{"Inherits", parent},
		{"Value", item.Value.String()},
	})
}

type ExprTable struct {
	OpenSpan  logger.Span
	Items     []*TableItem
	CloseSpan logger.Span

	// Used by interpreter.
	Cache map[string]Expr
}

func (table *ExprTable) Span() logger.Span {
	return logger.Span{
		Start: table.OpenSpan.Start,
		End:   table.CloseSpan.End,
	}
}

func (table *ExprTable) String() string {
	items := []string{}

	for _, item := range table.Items {
		items = append(items, item.String())
	}

	cache := map[string]string{}

	for key, cached := range table.Cache {
		cache[key] = cached.String()
	}

	return debugStruct("Table", []debugField{
		{"Items", debugSlice(items)},
		{"Cache", debugMap(cache)},
	})
}
