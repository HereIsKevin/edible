package evaluator

import (
	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
)

type refData struct {
	parent    parser.Expr
	value     parser.Expr
	evaluated bool
}

type Evaluator struct {
	expr     parser.Expr
	logger   *logger.Logger
	refDatas map[*parser.ExprRef]*refData
}

func New(expr parser.Expr, logger *logger.Logger) *Evaluator {
	return &Evaluator{
		expr:     expr,
		logger:   logger,
		refDatas: map[*parser.ExprRef]*refData{},
	}
}

func (evaluator *Evaluator) Evaluate() parser.Expr {
	if _, ok := evaluator.expr.(*parser.ExprTable); ok {
		evaluator.resolve(evaluator.expr, evaluator.expr)
	} else {
		evaluator.resolve(nil, evaluator.expr)
	}

	if err := evaluator.evaluate(evaluator.expr); err != nil {
		evaluator.logger.AddError(err)
	}

	return evaluator.expr
}

func (evaluator *Evaluator) resolve(parent parser.Expr, expr parser.Expr) {
	switch expr := expr.(type) {
	case *parser.ExprStr, *parser.ExprBool, *parser.ExprInt, *parser.ExprFloat:
		// Skip literals

	case *parser.ExprRef:
		evaluator.refDatas[expr] = &refData{
			parent:    parent,
			value:     nil,
			evaluated: false,
		}

		for _, key := range expr.Keys {
			evaluator.resolve(parent, key)
		}

	case *parser.ExprUnary:
		evaluator.resolve(parent, expr.Right)

	case *parser.ExprBinary:
		evaluator.resolve(parent, expr.Left)
		evaluator.resolve(parent, expr.Right)

	case *parser.ExprArray:
		for _, item := range expr.Items {
			evaluator.resolve(parent, item)
		}

	case *parser.ExprTable:
		for _, item := range expr.Items {
			evaluator.resolve(expr, item.Key)
			evaluator.resolve(expr, item.Value)

			if item.Parent != nil {
				evaluator.resolve(expr, item.Parent)
			}
		}
	}
}

func (evaluator *Evaluator) evaluate(expr parser.Expr) error {
	switch expr := expr.(type) {
	case *parser.ExprStr, *parser.ExprBool, *parser.ExprInt, *parser.ExprFloat:
		// Skip literals

	case *parser.ExprRef:
		if err := evaluator.evaluateRef(expr); err != nil {
			return err
		}

	case *parser.ExprUnary:
		panic("evaluateUnary is not implemented")

	case *parser.ExprBinary:
		panic("evaluateBinary is not implemented")

	case *parser.ExprArray:
		panic("evaluateArray is not implemented")

	case *parser.ExprTable:
		if err := evaluator.evaluateTable(expr); err != nil {
			return err
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateRef(ref *parser.ExprRef) error {
	data := evaluator.refDatas[ref]

	// Do not evaluate already evaluated references.
	if data.evaluated {
		return nil
	}

	// Set status before starting to prevent infinite recursion.
	data.evaluated = true

	// Choose base expression based on modifier.
	expr := evaluator.expr
	if ref.Modifier == parser.RefRelative {
		expr = data.parent
	}

loop:
	for _, rawKey := range ref.Keys {
		// Unwrap all references in the current key.
		keyExpr, err := evaluator.unwrap(rawKey)
		if err != nil {
			return err
		}

		// Unwrap all references in the current value.
		currentExpr, err := evaluator.unwrap(expr)
		if err != nil {
			return err
		}

		switch current := currentExpr.(type) {
		case *parser.ExprArray:
			panic("ExprArray is not implemented for evaluateRef")

		case *parser.ExprTable:
			// Make sure the key is a string.
			key, ok := keyExpr.(*parser.ExprStr)
			if !ok {
				return &logger.Error{
					Message: "Expect string for table key.",
					Span:    rawKey.Span(),
				}
			}

			// TODO: Use map for better performance.
			for _, item := range current.Items {
				// Attempt to evaluate key from table, skip on failure.
				itemKeyExpr, err := evaluator.unwrap(item.Key)
				if err != nil {
					continue
				}

				// Make sure the item key is a string.
				itemKey, ok := itemKeyExpr.(*parser.ExprStr)
				if !ok {
					continue
				}

				// Go on to the next iteration if the keys match.
				if key.Value == itemKey.Value {
					expr = item.Value
					continue loop
				}
			}

			// Completed loop means none of the keys match.
			return &logger.Error{
				Message: "Key not found.",
				Span:    rawKey.Span(),
			}

		default:
			return &logger.Error{
				Message: "Expect array or table.",
				Span:    expr.Span(),
			}
		}
	}

	data.value = expr

	return nil
}

func (evaluator *Evaluator) evaluateTable(table *parser.ExprTable) error {
	for _, item := range table.Items {
		if err := evaluator.evaluate(item.Key); err != nil {
			return err
		}

		key, err := evaluator.unwrap(item.Key)
		if err != nil {
			return err
		}

		item.Key = key

		// TODO: Evaluate parents.

		if err := evaluator.evaluate(item.Value); err != nil {
			return err
		}

		value, err := evaluator.unwrap(item.Value)
		if err != nil {
			return err
		}

		item.Value = value
	}

	return nil
}

func (evaluator *Evaluator) unwrap(expr parser.Expr) (parser.Expr, error) {
	for {
		switch current := expr.(type) {
		case *parser.ExprStr,
			*parser.ExprBool,
			*parser.ExprInt,
			*parser.ExprFloat,
			*parser.ExprArray,
			*parser.ExprTable:

			// Exit if the expression is a concrete value.
			return expr, nil

		case *parser.ExprRef:
			// Evaluate the reference.
			evaluator.evaluate(current)

			// Extract the data within reference.
			data := evaluator.refDatas[current]
			if data.value == nil {
				return nil, &logger.Error{
					Message: "Failed to unwrap reference.",
					Span:    expr.Span(),
				}
			}

			// Repeat with expression from reference.
			expr = data.value

		case *parser.ExprUnary:
			panic("ExprUnary is not implemented for evaluateKey")

		case *parser.ExprBinary:
			panic("ExprBinary is not implemented for evaluateKey")
		}
	}
}
