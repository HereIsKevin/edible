package evaluator

import (
	"fmt"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
)

type Evaluator struct {
	expr   parser.Expr
	logger *logger.Logger
	value  any

	root   *Evaluator
	parent *Evaluator
}

func New(expr parser.Expr, logger *logger.Logger) *Evaluator {
	evaluator := &Evaluator{
		expr:   expr,
		logger: logger,
		value:  nil,
	}

	// Root of root is itself.
	evaluator.root = evaluator

	// Parent of root is parent if it is a table since parent always refers to the
	// closest table up the hierarchy, which is itself for a root table.
	_, ok := evaluator.expr.(*parser.ExprTable)
	if ok {
		evaluator.parent = evaluator
	}

	return evaluator
}

func (evaluator *Evaluator) Evaluate() {
	evaluator.evaluateStatic()
}

func (evaluator *Evaluator) evaluateStatic() {
	switch expr := evaluator.expr.(type) {
	// Literal expressions
	case *parser.ExprStr:
		evaluator.value = expr.Value
	case *parser.ExprBool:
		evaluator.value = expr.Value
	case *parser.ExprInt:
		evaluator.value = expr.Value
	case *parser.ExprFloat:
		evaluator.value = expr.Value

	// Dynamic expressions
	case *parser.ExprRef:
		// References are always dynamic.
	case *parser.ExprUnary:
		// Unary operations are often dynamic and are not containers, so it is not worth
		// it to evaluate the possibly static ones.
	case *parser.ExprBinary:
		// Similarly, binary operations are often dynamic, so it is not worth it to
		// evaluate them.

	// Container expressions
	case *parser.ExprArray:
		evaluator.evaluateStaticArray(expr)
	case *parser.ExprTable:
		evaluator.evaluateStaticTable(expr)
	}
}

func (evaluator *Evaluator) evaluateStaticArray(array *parser.ExprArray) {
	evaluators := []*Evaluator{}

	for index, item := range array.Items {
		switch item.(type) {
		case *parser.ExprStr,
			*parser.ExprBool,
			*parser.ExprInt,
			*parser.ExprFloat,
			*parser.ExprArray,
			*parser.ExprTable:

			// Evaluate literals and containers as static items.
			itemEvaluator := New(item, evaluator.logger)
			itemEvaluator.root = evaluator.root
			itemEvaluator.parent = evaluator.parent
			itemEvaluator.evaluateStatic()

			// Add the evaluator.
			evaluators = append(evaluators, itemEvaluator)

			// Mark the item as completed.
			array.Items[index] = nil

		case *parser.ExprRef, *parser.ExprUnary, *parser.ExprBinary:
			// Skip over dynamic items.
			evaluators = append(evaluators, nil)
		}
	}

	evaluator.value = evaluators
}

func (evaluator *Evaluator) evaluateStaticTable(table *parser.ExprTable) {
	evaluators := map[string]*Evaluator{}

	for index, item := range table.Items {
		// Only evaluate accept static string keys.
		key, ok := item.Key.(*parser.ExprStr)
		if !ok {
			continue
		}

		// Only evaluate items without parents.
		if item.Inherits != nil {
			continue
		}

		switch item.Value.(type) {
		case *parser.ExprStr,
			*parser.ExprBool,
			*parser.ExprInt,
			*parser.ExprFloat,
			*parser.ExprArray,
			*parser.ExprTable:

			// Evaluate literals and containers as static items.
			valueEvaluator := New(item.Value, evaluator.logger)
			valueEvaluator.root = evaluator.root
			valueEvaluator.parent = evaluator.parent
			valueEvaluator.evaluateStatic()

			// Add the evaluator.
			evaluators[key.Value] = valueEvaluator

			// Mark the item as completed.
			table.Items[index] = nil

		case *parser.ExprRef, *parser.ExprUnary, *parser.ExprBinary:
			// Skip over dynamic items.
		}
	}

	evaluator.value = evaluators
}

func (evaluator *Evaluator) String() string {
	switch value := evaluator.value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", value)
	case bool:
		return fmt.Sprintf("%t", value)
	case int64:
		return fmt.Sprintf("%d", value)
	case float64:
		return fmt.Sprintf("%f", value)

	case []*Evaluator:
		debugSlice := []string{}

		for _, item := range value {
			debugSlice = append(debugSlice, item.String())
		}

		return logger.DebugSlice(debugSlice)

	case map[string]*Evaluator:
		debugMap := map[string]string{}

		for key, value := range value {
			debugMap[key] = value.String()
		}

		return logger.DebugMap(debugMap)

	default:
		return "Unknown"
	}
}
