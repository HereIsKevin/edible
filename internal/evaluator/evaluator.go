package evaluator

import (
	"fmt"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
)

type Evaluator struct {
	expr      parser.Expr
	logger    *logger.Logger
	evaluated any

	root   *Evaluator
	parent *Evaluator
}

func New(expr parser.Expr, logger *logger.Logger) *Evaluator {
	evaluator := &Evaluator{
		expr:      expr,
		logger:    logger,
		evaluated: nil,

		root:   nil,
		parent: nil,
	}

	evaluator.root = evaluator
	return evaluator
}

func (evaluator *Evaluator) Evaluate() any {
	if evaluator.evaluated != nil {
		return evaluator.evaluated
	}

	switch expr := evaluator.expr.(type) {
	case *parser.ExprStr:
		evaluator.evaluated = expr.Value
	case *parser.ExprBool:
		evaluator.evaluated = expr.Value
	case *parser.ExprInt:
		evaluator.evaluated = expr.Value
	case *parser.ExprFloat:
		evaluator.evaluated = expr.Value

	case *parser.ExprRef:
		evaluator.evaluateRef(expr)
	case *parser.ExprArray:
		evaluator.evaluateArray(expr)
	case *parser.ExprTable:
		evaluator.evaluateTable(expr)
	}

	return evaluator.evaluated
}

func (evaluator *Evaluator) evaluateRef(ref *parser.ExprRef) {
	current := evaluator.root
	if ref.Modifier == parser.RefRelative {
		current = evaluator.parent
	}

	for _, key := range ref.Keys {
		keyEvaluator := New(key, evaluator.logger)
		keyEvaluator.root = evaluator.root
		keyEvaluator.parent = evaluator.parent

		if current == nil {
			// ERROR
			return
		}

		switch key := keyEvaluator.Evaluate().(type) {
		case string:
			current = current.key(key)
		case int64:
			current = current.index(int(key))
		default:
			// ERROR
			return
		}
	}

	evaluator.evaluated = current
}

func (evaluator *Evaluator) evaluateArray(array *parser.ExprArray) {
	evaluated := []*Evaluator{}

	for _, item := range array.Items {
		itemEvaluator := New(item, evaluator.logger)
		itemEvaluator.root = evaluator.root
		itemEvaluator.parent = evaluator.parent
		itemEvaluator.Evaluate()

		evaluated = append(evaluated, itemEvaluator)
	}

	evaluator.evaluated = evaluated
}

func (evaluator *Evaluator) evaluateTable(table *parser.ExprTable) {
	evaluated := map[string]*Evaluator{}

	for _, item := range table.Items {
		keyEvaluator := New(item.Key, evaluator.logger)
		keyEvaluator.root = evaluator.root
		keyEvaluator.parent = evaluator
		key, ok := keyEvaluator.Evaluate().(string)
		if !ok {
			// ERROR
			return
		}

		valueEvaluator := New(item.Value, evaluator.logger)
		valueEvaluator.root = evaluator.root
		valueEvaluator.parent = evaluator
		valueEvaluator.Evaluate()

		evaluated[key] = valueEvaluator
	}

	evaluator.evaluated = evaluated
}

func (evaluator *Evaluator) key(key string) *Evaluator {
	table, ok := evaluator.Evaluate().(map[string]*Evaluator)
	if !ok {
		return nil
	}

	return table[key]
}

func (evaluator *Evaluator) index(index int) *Evaluator {
	array, ok := evaluator.Evaluate().([]*Evaluator)
	if !ok {
		return nil
	}

	if len(array) > index {
		return array[index]
	}

	return nil
}

func (evaluator *Evaluator) String() string {
	switch value := evaluator.evaluated.(type) {
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

// func (evaluator *Evaluator) span() logger.Span {
// 	return evaluator.expr.Span()
// }
