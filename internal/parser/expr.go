package parser

import (
	"fmt"

	"github.com/HereIsKevin/edible/internal/logger"
)

type Expr interface {
	Pos() logger.Pos
	fmt.Stringer
}

// String

type ExprStr struct {
	Value    string
	Position logger.Pos
}

func (str *ExprStr) Pos() logger.Pos {
	return str.Position
}

func (str *ExprStr) String() string {
	return fmt.Sprintf("Str(\"%s\")", str.Value)
}

// Boolean

type ExprBool struct {
	Value    bool
	Position logger.Pos
}

func (bool *ExprBool) Pos() logger.Pos {
	return bool.Position
}

func (bool *ExprBool) String() string {
	return fmt.Sprintf("Bool(%t)", bool.Value)
}

// Integer

type ExprInt struct {
	Value    int64
	Position logger.Pos
}

func (int *ExprInt) Pos() logger.Pos {
	return int.Position
}

func (int *ExprInt) String() string {
	return fmt.Sprintf("Int(%d)", int.Value)
}

// Float

type ExprFloat struct {
	Value    float64
	Position logger.Pos
}

func (float *ExprFloat) Pos() logger.Pos {
	return float.Position
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
	Modifier RefModifier
	Keys     []Expr
	Position logger.Pos
}

func (ref *ExprRef) Pos() logger.Pos {
	return ref.Position
}

func (ref *ExprRef) String() string {
	keys := []string{}

	for _, key := range ref.Keys {
		keys = append(keys, key.String())
	}

	return logger.DebugStruct("Ref", []logger.DebugField{
		{Key: "Modifier", Value: ref.Modifier.String()},
		{Key: "Keys", Value: logger.DebugSlice(keys)},
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
	Op       UnaryOp
	Right    Expr
	Position logger.Pos
}

func (unary *ExprUnary) Pos() logger.Pos {
	return unary.Position
}

func (unary *ExprUnary) String() string {
	return logger.DebugStruct("Unary", []logger.DebugField{
		{Key: "Op", Value: unary.Op.String()},
		{Key: "Right", Value: unary.Right.String()},
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
	Left     Expr
	Op       BinaryOp
	Right    Expr
	Position logger.Pos
}

func (binary *ExprBinary) Pos() logger.Pos {
	return binary.Position
}

func (binary *ExprBinary) String() string {
	return logger.DebugStruct("Binary", []logger.DebugField{
		{Key: "Left", Value: binary.Left.String()},
		{Key: "Op", Value: binary.Op.String()},
		{Key: "Right", Value: binary.Right.String()},
	})
}

// Array

type ExprArray struct {
	Items    []Expr
	Position logger.Pos
}

func (array *ExprArray) Pos() logger.Pos {
	return array.Position
}

func (array *ExprArray) String() string {
	items := []string{}

	for _, item := range array.Items {
		items = append(items, item.String())
	}

	return logger.DebugStruct("Array", []logger.DebugField{
		{Key: "Items", Value: logger.DebugSlice(items)},
	})
}

// Table

type TableItem struct {
	Key    *ExprStr
	Parent Expr
	Value  Expr
}

func (item *TableItem) String() string {
	parent := "nil"

	if item.Parent != nil {
		parent = item.Parent.String()
	}

	return logger.DebugStruct("", []logger.DebugField{
		{Key: "Key", Value: item.Key.String()},
		{Key: "Parent", Value: parent},
		{Key: "Value", Value: item.Value.String()},
	})
}

type ExprTable struct {
	Items    []*TableItem
	Position logger.Pos
}

func (table *ExprTable) Pos() logger.Pos {
	return table.Position
}

func (table *ExprTable) String() string {
	items := []string{}

	for _, item := range table.Items {
		items = append(items, item.String())
	}

	return logger.DebugStruct("Table", []logger.DebugField{
		{Key: "Items", Value: logger.DebugSlice(items)},
	})
}
