// TODO: Check for potential issues with nil.

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

type arrayDataItem struct {
	value     parser.Expr
	evaluated bool
}

type arrayData struct {
	value     []*arrayDataItem
	evaluated bool
}

type tableDataItem struct {
	parent    parser.Expr
	value     parser.Expr
	evaluated bool
}

type tableData struct {
	value     map[string]*tableDataItem
	evaluated bool
}

type Evaluator struct {
	expr   parser.Expr
	logger *logger.Logger

	refDatas    map[*parser.ExprRef]*refData
	unaryDatas  map[*parser.ExprUnary]*opData
	binaryDatas map[*parser.ExprBinary]*opData
	arrayDatas  map[*parser.ExprArray]*arrayData
	tableDatas  map[*parser.ExprTable]*tableData
}

func New(expr parser.Expr, logger *logger.Logger) *Evaluator {
	return &Evaluator{
		expr:   expr,
		logger: logger,

		refDatas:    map[*parser.ExprRef]*refData{},
		unaryDatas:  map[*parser.ExprUnary]*opData{},
		binaryDatas: map[*parser.ExprBinary]*opData{},
		arrayDatas:  map[*parser.ExprArray]*arrayData{},
		tableDatas:  map[*parser.ExprTable]*tableData{},
	}
}

func (evaluator *Evaluator) Evaluate() any {
	if _, ok := evaluator.expr.(*parser.ExprTable); ok {
		evaluator.bind(evaluator.expr, evaluator.expr)
	} else {
		evaluator.bind(evaluator.expr, nil)
	}

	if err := evaluator.evaluate(evaluator.expr); err != nil {
		evaluator.logger.AddError(err)
	}

	return evaluator.resolve(evaluator.expr)
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
		evaluator.arrayDatas[current] = &arrayData{
			value:     []*arrayDataItem{},
			evaluated: false,
		}

		for _, item := range current.Items {
			evaluator.bind(item, parent)
		}

	case *parser.ExprTable:
		evaluator.tableDatas[current] = &tableData{
			value:     map[string]*tableDataItem{},
			evaluated: false,
		}

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

	case *parser.ExprRef:
		if err := evaluator.evaluateRef(current); err != nil {
			return err
		}

	case *parser.ExprUnary:
		panic("evaluateUnary is not implemented")

	case *parser.ExprBinary:
		panic("evaluateBinary is not implemented")

	case *parser.ExprArray:
		if err := evaluator.evaluateArray(current); err != nil {
			return err
		}

	case *parser.ExprTable:
		if err := evaluator.evaluateTable(current); err != nil {
			return err
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateRef(ref *parser.ExprRef) error {
	data := evaluator.refDatas[ref]

	// Exit if reference is already evaluated.
	if data.evaluated {
		return nil
	}

	// Mark reference as evaluated.
	data.evaluated = true

	// Start from the root expression.
	expr := data.root

	for _, rawKey := range ref.Keys {
		// Unwrap the current key.
		keyExpr, err := evaluator.unwrap(rawKey)
		if err != nil {
			return err
		}

		// Unwrap the current value.
		currentExpr, err := evaluator.unwrap(expr)
		if err != nil {
			return err
		}

		switch current := currentExpr.(type) {
		case *parser.ExprArray:
			// Make sure the index is a integer.
			index, ok := keyExpr.(*parser.ExprInt)
			if !ok {
				return &logger.Error{
					Message: "Expect integer for array index.",
					Pos:     rawKey.Pos(),
				}
			}

			// Make sure array indices are evaluated.
			evaluator.evaluateArrayIndices(current)

			array := evaluator.arrayDatas[current].value
			indexInner := int(index.Value)

			// Make sure index is in bounds.
			if len(array) <= indexInner {
				return &logger.Error{
					Message: "Index out of bounds.",
					Pos:     rawKey.Pos(),
				}
			}

			// Evaluate the item.
			if err := evaluator.evaluateArrayValue(current, indexInner); err != nil {
				return err
			}

			// Repeat with the item value.
			expr = array[indexInner].value

		// TODO: Prove that the initial table is already merged.
		case *parser.ExprTable:
			// Make sure the key is a string.
			key, ok := keyExpr.(*parser.ExprStr)
			if !ok {
				return &logger.Error{
					Message: "Expect string for table key.",
					Pos:     rawKey.Pos(),
				}
			}

			// Make sure the table keys are evaluated.
			if err := evaluator.evaluateTableKeys(current); err != nil {
				return err
			}

			// Get the item from the table.
			item, ok := evaluator.tableDatas[current].value[key.Value]
			if !ok {
				return &logger.Error{
					Message: "Key not found.",
					Pos:     rawKey.Pos(),
				}
			}

			// Evaluate the item.
			if err := evaluator.evaluateTableValue(current, key.Value); err != nil {
				return err
			}

			// Repeat with the item value.
			expr = item.value
		}
	}

	// Unwrap the value.
	value, err := evaluator.unwrap(expr)
	if err != nil {
		return err
	}

	data.value = value

	return nil
}

func (evaluator *Evaluator) evaluateUnary(unary *parser.ExprUnary) error {
	return &logger.Error{
		Message: "Unary expressions are not supported.",
		Pos:     unary.Pos(),
	}
}

func (evaluator *Evaluator) evaluateBinary(binary *parser.ExprBinary) error {
	return &logger.Error{
		Message: "Binary expressions are not supported.",
		Pos:     binary.Pos(),
	}
}

func (evaluator *Evaluator) evaluateArrayIndices(array *parser.ExprArray) {
	data := evaluator.arrayDatas[array]

	// Exit if indices are already evaluated.
	if data.evaluated {
		return
	}

	// Mark indices as evaluated.
	data.evaluated = true

	for _, item := range array.Items {
		// Create entry in array.
		data.value = append(data.value, &arrayDataItem{
			value:     item,
			evaluated: false,
		})
	}
}

func (evaluator *Evaluator) evaluateArrayValue(
	array *parser.ExprArray,
	index int,
) error {
	data := evaluator.arrayDatas[array]
	item := data.value[index]

	// Exit if item is already evaluated.
	if item.evaluated {
		return nil
	}

	// Mark item as evaluated.
	item.evaluated = true

	// Evaluate value
	if err := evaluator.evaluate(item.value); err != nil {
		return err
	}

	// Unwrap value.
	valueExpr, err := evaluator.unwrap(item.value)
	if err != nil {
		return err
	}

	item.value = valueExpr

	return nil
}

func (evaluator *Evaluator) evaluateArray(array *parser.ExprArray) error {
	evaluator.evaluateArrayIndices(array)

	for index := range evaluator.arrayDatas[array].value {
		if err := evaluator.evaluateArrayValue(array, index); err != nil {
			return err
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateTableKeys(table *parser.ExprTable) error {
	data := evaluator.tableDatas[table]

	// Exit if keys are already evaluated.
	if data.evaluated {
		return nil
	}

	// Mark keys as evaluated.
	data.evaluated = true

	for _, item := range table.Items {
		// Check for duplicate keys.
		if _, ok := data.value[item.Key.Value]; ok {
			return &logger.Error{
				Message: "Duplicate key in table.",
				Pos:     item.Key.Pos(),
			}
		}

		// Create entry in table.
		data.value[item.Key.Value] = &tableDataItem{
			parent:    item.Parent,
			value:     item.Value,
			evaluated: false,
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateTableValue(
	table *parser.ExprTable,
	key string,
) error {
	data := evaluator.tableDatas[table]
	item := data.value[key]

	// Exit if item is already evaluated.
	if item.evaluated {
		return nil
	}

	// Mark item as evaluated.
	item.evaluated = true

	// Evaluate parent.
	if err := evaluator.evaluate(item.parent); err != nil {
		return err
	}

	// Unwrap parent.
	parentExpr, err := evaluator.unwrap(item.parent)
	if err != nil {
		return err
	}

	item.parent = parentExpr

	// Evaluate value.
	if err := evaluator.evaluate(item.value); err != nil {
		return err
	}

	// Unwrap value.
	valueExpr, err := evaluator.unwrap(item.value)
	if err != nil {
		return err
	}

	item.value = valueExpr

	// Make sure the value is a table.
	value, ok := item.value.(*parser.ExprTable)
	if !ok {
		return nil
	}

	// Make sure the parent is a table.
	parent, ok := item.parent.(*parser.ExprTable)
	if !ok {
		return nil
	}

	evaluator.evaluateTableKeys(value)
	evaluator.evaluateTableKeys(parent)

	valueData := evaluator.tableDatas[value]
	parentData := evaluator.tableDatas[parent]

	for key, item := range parentData.value {
		// Merge item from parent if key is not already there.
		if _, ok := valueData.value[key]; !ok {
			valueData.value[key] = item
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateTable(table *parser.ExprTable) error {
	if err := evaluator.evaluateTableKeys(table); err != nil {
		return err
	}

	for key := range evaluator.tableDatas[table].value {
		if err := evaluator.evaluateTableValue(table, key); err != nil {
			return err
		}
	}

	return nil
}

// TODO: Prove that only a single unwrap is ever needed.
func (evaluator *Evaluator) unwrap(expr parser.Expr) (parser.Expr, error) {
	for {
		if expr == nil {
			return nil, nil
		}

		switch current := expr.(type) {
		case *parser.ExprStr,
			*parser.ExprBool,
			*parser.ExprInt,
			*parser.ExprFloat,
			*parser.ExprArray,
			*parser.ExprTable:

			// Exit on concrete values.
			return expr, nil

		case *parser.ExprRef:
			// Evaluate the reference.
			if err := evaluator.evaluateRef(current); err != nil {
				return nil, err
			}

			// Repeat with expression from reference.
			expr = evaluator.refDatas[current].value

		case *parser.ExprUnary:
			// Evaluate the unary.
			if err := evaluator.evaluateUnary(current); err != nil {
				return nil, err
			}

			// Repeat with expression from unary.
			expr = evaluator.unaryDatas[current].value

		case *parser.ExprBinary:
			// Evaluate the binary.
			if err := evaluator.evaluateBinary(current); err != nil {
				return nil, err
			}

			// Repeat with expression from binary.
			expr = evaluator.binaryDatas[current].value
		}
	}
}

// TODO: Prove that unwrap and errors are not necessary.
func (evaluator *Evaluator) resolve(expr parser.Expr) any {
	expr, err := evaluator.unwrap(expr)
	if err != nil {
		panic("Unable to resolve expression.")
	}

	switch current := expr.(type) {
	case *parser.ExprStr:
		return current.Value

	case *parser.ExprBool:
		return current.Value

	case *parser.ExprInt:
		return current.Value

	case *parser.ExprFloat:
		return current.Value

	case *parser.ExprArray:
		items := []any{}

		for _, item := range evaluator.arrayDatas[current].value {
			items = append(items, evaluator.resolve(item.value))
		}

		return items

	case *parser.ExprTable:
		items := map[string]any{}

		for key, item := range evaluator.tableDatas[current].value {
			items[key] = evaluator.resolve(item.value)
		}

		return items

	case *parser.ExprRef, *parser.ExprUnary, *parser.ExprBinary:
		panic("Failed to resolve expression.")
	}

	return nil
}
