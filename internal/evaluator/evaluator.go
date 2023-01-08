package evaluator

import (
	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
)

type refData struct {
	root      parser.Expr
	value     parser.Expr
	evaluated bool
}

type opData struct {
	value     parser.Expr
	evaluated bool
}

type tableDataValue struct {
	parent    parser.Expr
	value     parser.Expr
	evaluated bool
}

type tableData struct {
	value     map[string]*tableDataValue
	evaluated map[int]bool
}

type Evaluator struct {
	expr   parser.Expr
	logger *logger.Logger

	refDatas    map[*parser.ExprRef]*refData
	unaryDatas  map[*parser.ExprUnary]*opData
	binaryDatas map[*parser.ExprBinary]*opData
	tableDatas  map[*parser.ExprTable]*tableData
}

func New(expr parser.Expr, logger *logger.Logger) *Evaluator {
	return &Evaluator{
		expr:   expr,
		logger: logger,

		refDatas:    map[*parser.ExprRef]*refData{},
		unaryDatas:  map[*parser.ExprUnary]*opData{},
		binaryDatas: map[*parser.ExprBinary]*opData{},
		tableDatas:  map[*parser.ExprTable]*tableData{},
	}
}

func (evaluator *Evaluator) bind(expr parser.Expr, parent parser.Expr) {
	switch current := expr.(type) {
	case *parser.ExprStr, *parser.ExprBool, *parser.ExprInt, *parser.ExprFloat:
		// Skip literals.

	case *parser.ExprRef:
		root := parent
		if current.Modifier == parser.RefAbsolute {
			root = evaluator.expr
		}

		evaluator.refDatas[current] = &refData{
			root:      root,
			value:     nil,
			evaluated: false,
		}

		for _, key := range current.Keys {
			evaluator.bind(key, parent)
		}

	case *parser.ExprUnary:
		evaluator.unaryDatas[current] = &opData{
			value:     nil,
			evaluated: false,
		}

		evaluator.bind(current.Right, parent)

	case *parser.ExprBinary:
		evaluator.binaryDatas[current] = &opData{
			value:     nil,
			evaluated: false,
		}

		evaluator.bind(current.Left, parent)
		evaluator.bind(current.Right, parent)

	case *parser.ExprArray:
		for _, item := range current.Items {
			evaluator.bind(item, parent)
		}

	case *parser.ExprTable:
		for _, item := range current.Items {
			evaluator.bind(item.Key, parent)
			evaluator.bind(item.Value, parent)

			if item.Parent != nil {
				evaluator.bind(item.Parent, parent)
			}
		}
	}
}

func (evaluator *Evaluator) evaluate(expr parser.Expr) error {
	switch current := expr.(type) {
	case *parser.ExprStr, *parser.ExprBool, *parser.ExprInt, *parser.ExprFloat:
		// Skip literals.
		return nil

	case *parser.ExprRef:
		return evaluator.evaluateRef(current)

	case *parser.ExprUnary:
		return evaluator.evaluateUnary(current)

	case *parser.ExprBinary:
		return evaluator.evaluateBinary(current)

	case *parser.ExprArray:
		return evaluator.evaluateArray(current)

	case *parser.ExprTable:
		return evaluator.evaluateTable(current)
	}
}

func (evaluator *Evaluator) evaluateRef(ref *parser.ExprRef) error {

}

func (evaluator *Evaluator) evaluateUnary(unary *parser.ExprUnary) error {

}

func (evaluator *Evaluator) evaluateBinary(binary *parser.ExprBinary) error {

}

func (evaluator *Evaluator) evaluateArray(array *parser.ExprArray) error {

}

func (evaluator *Evaluator) evaluateTableKeys(table *parser.ExprTable) error {
	for index := range table.Items {
		evaluator.evaluateTableKey(table, index)
	}
}

func (evaluator *Evaluator) evaluateTableKey(
	table *parser.ExprTable,
	index int,
) (*parser.ExprStr, error) {
	data := evaluator.tableDatas[table]
	item := table.Items[index]

	// Unwrap the table key.
	keyExpr, err := evaluator.unwrap(item.Key)
	if err != nil {
		return nil, err
	}

	// Make sure it is a string.
	key, ok := keyExpr.(*parser.ExprStr)
	if !ok {
		return nil, &logger.Error{
			Message: "Expect string for table key.",
			Span:    item.Key.Span(),
		}
	}

	// Mark the key as evaluated.
	data.evaluated[index] = true

	// Create an entry for value to be evaluated.
	data.value[key.Value] = &tableDataValue{
		parent:    item.Parent,
		value:     item.Value,
		evaluated: false,
	}

	return key, nil
}

func (evaluator *Evaluator) evaluateTableMerge(table *parser.ExprTable, key string) error {
	data := evaluator.tableDatas[table]
	value := data.value[key]

	// Make sure the value is a table.
	valueExpr, ok := value.value.(*parser.ExprTable)
	if !ok {
		return nil
	}

	// Make sure the parent is a table.
	parentExpr, ok := value.parent.(*parser.ExprTable)
	if !ok {
		return nil
	}

	valueData := evaluator.tableDatas[valueExpr]
	parentData := evaluator.tableDatas[parentExpr]

	for index := range parentExpr.Items {
		evaluator.evaluateTableKey(parentExpr, index)
	}

	for index := range valueExpr.Items {
		evaluator.evaluateTableKey(valueExpr, index)
	}

	for key, value := range parentData.value {
		key.
	}
}

func (evaluator *Evaluator) evaluateTableValue(table *parser.ExprTable, key string) error {
	data := evaluator.tableDatas[table]
	dataValue := data.value[key]
	dataValue.evaluated = true

	if err := evaluator.evaluate(dataValue.parent); err != nil {
		return err
	}

	if err := evaluator.evaluate(dataValue.value); err != nil {
		return err
	}

	evaluator.merge(valueExpr, parentExpr)
	return nil
}

func (evaluator *Evaluator) evaluateTable(table *parser.ExprTable) error {
	for _, item := range table.Items {
		key, err := evaluator.unwrap(item.Key)
		// if err !=
	}
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

			// Extract value from data.
			data := evaluator.refDatas[current]

			// Make sure there is an expression in the data.
			if data.value == nil {
				return nil, &logger.Error{
					Message: "Failed to unwrap reference.",
					Span:    expr.Span(),
				}
			}

			// Repeat with the expression.
			expr = data.value

		case *parser.ExprUnary:
			// Evaluate the unary expression.
			evaluator.evaluate(current)

			// Extract value from data.
			data := evaluator.unaryDatas[current]

			// Make sure there is an expression in the data.
			if data.value == nil {
				return nil, &logger.Error{
					Message: "Failed to unwrap unary expression.",
					Span:    expr.Span(),
				}
			}

			// Repeat with the expression.
			expr = data.value

		case *parser.ExprBinary:
			// Evaluate the binary expression.
			evaluator.evaluate(current)

			// Extract value from data.
			data := evaluator.binaryDatas[current]

			// Make sure there is an expression in the data.
			if data.value == nil {
				return nil, &logger.Error{
					Message: "Failed to unwrap binary expression.",
					Span:    expr.Span(),
				}
			}

			// Repeat with the expression.
			expr = data.value
		}
	}
}
