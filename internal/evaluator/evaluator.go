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

type itemData struct {
	value     parser.Expr
	evaluated bool
}

type arrayData struct {
	items     []*itemData
	evaluated bool
}

type tableData struct {
	items     map[string]*itemData
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

	value, err := evaluator.resolve(evaluator.expr)
	if err != nil {
		evaluator.logger.AddError(err)
	}

	return value
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
			items:     []*itemData{},
			evaluated: false,
		}

		for _, item := range current.Items {
			evaluator.bind(item, parent)
		}

	case *parser.ExprTable:
		evaluator.tableDatas[current] = &tableData{
			items:     map[string]*itemData{},
			evaluated: false,
		}

		if current.Parent != nil {
			evaluator.bind(current.Parent, current)
		}

		for _, item := range current.Items {
			evaluator.bind(item.Key, current)
			evaluator.bind(item.Value, current)
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
		if err := evaluator.evaluateUnary(current); err != nil {
			return err
		}

	case *parser.ExprBinary:
		if err := evaluator.evaluateBinary(current); err != nil {
			return err
		}

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

	// Start from root expression.
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
			// Make sure the key is an integer.
			index, ok := keyExpr.(*parser.ExprInt)
			if !ok {
				return &logger.Error{
					Message: "Expect integer for array index.",
					Pos:     rawKey.Pos(),
				}
			}

			// Make sure array indices are evaluated.
			evaluator.evaluateArrayIndices(current)

			items := evaluator.arrayDatas[current].items

			// Make sure index is in bounds.
			if int64(len(items)) <= index.Value {
				return &logger.Error{
					Message: "Index out of bounds.",
					Pos:     rawKey.Pos(),
				}
			}

			item := items[index.Value]

			// Evaluate the item.
			if err := evaluator.evaluateItem(item); err != nil {
				return err
			}

			// Repeat with the item value.
			expr = item.value

		case *parser.ExprTable:
			// Make sure the key is a string.
			key, ok := keyExpr.(*parser.ExprStr)
			if !ok {
				return &logger.Error{
					Message: "Expect string for table key.",
					Pos:     rawKey.Pos(),
				}
			}

			// Make sure table keys are evaluated.
			if err := evaluator.evaluateTableKeys(current); err != nil {
				return err
			}

			// Get item from table if possible.
			item, ok := evaluator.tableDatas[current].items[key.Value]
			if !ok {
				return &logger.Error{
					Message: "Key not found.",
					Pos:     rawKey.Pos(),
				}
			}

			// Evalute the item.
			if err := evaluator.evaluateItem(item); err != nil {
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
	data := evaluator.unaryDatas[unary]

	// Exit if already evaluated.
	if data.evaluated {
		return nil
	}

	// Unwrap expression.
	expr, err := evaluator.unwrap(unary.Right)
	if err != nil {
		return err
	}

	switch unary.Op {
	case parser.UnaryPlus:
		switch current := expr.(type) {
		case *parser.ExprInt:
			data.value = &parser.ExprInt{
				Value:    current.Value,
				Position: unary.Right.Pos(),
			}

		case *parser.ExprFloat:
			data.value = &parser.ExprFloat{
				Value:    current.Value,
				Position: unary.Right.Pos(),
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     unary.Right.Pos(),
			}
		}

	case parser.UnaryMinus:
		switch current := expr.(type) {
		case *parser.ExprInt:
			data.value = &parser.ExprInt{
				Value:    -current.Value,
				Position: unary.Right.Pos(),
			}

		case *parser.ExprFloat:
			data.value = &parser.ExprFloat{
				Value:    -current.Value,
				Position: unary.Right.Pos(),
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     unary.Right.Pos(),
			}
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateBinary(binary *parser.ExprBinary) error {
	data := evaluator.binaryDatas[binary]

	// Exit if already evaluated.
	if data.evaluated {
		return nil
	}

	// Unwrap left.
	leftExpr, err := evaluator.unwrap(binary.Left)
	if err != nil {
		return err
	}

	// Unwrap right.
	rightExpr, err := evaluator.unwrap(binary.Right)
	if err != nil {
		return err
	}

	position := logger.Pos{
		Start: binary.Left.Pos().Start,
		End:   binary.Right.Pos().End,
		Line:  binary.Left.Pos().Line,
	}

	switch binary.Op {
	case parser.BinaryPlus:
		switch left := leftExpr.(type) {
		case *parser.ExprInt:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprInt{
					Value:    left.Value + right.Value,
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    float64(left.Value) + right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		case *parser.ExprFloat:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprFloat{
					Value:    left.Value + float64(right.Value),
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    left.Value + right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     binary.Left.Pos(),
			}
		}

	case parser.BinaryMinus:
		switch left := leftExpr.(type) {
		case *parser.ExprInt:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprInt{
					Value:    left.Value - right.Value,
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    float64(left.Value) - right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		case *parser.ExprFloat:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprFloat{
					Value:    left.Value - float64(right.Value),
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    left.Value - right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     binary.Left.Pos(),
			}
		}

	case parser.BinaryStar:
		switch left := leftExpr.(type) {
		case *parser.ExprInt:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprInt{
					Value:    left.Value * right.Value,
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    float64(left.Value) * right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		case *parser.ExprFloat:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprFloat{
					Value:    left.Value * float64(right.Value),
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    left.Value * right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     binary.Left.Pos(),
			}
		}

	case parser.BinarySlash:
		switch left := leftExpr.(type) {
		case *parser.ExprInt:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprInt{
					Value:    left.Value / right.Value,
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    float64(left.Value) / right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		case *parser.ExprFloat:
			switch right := rightExpr.(type) {
			case *parser.ExprInt:
				data.value = &parser.ExprFloat{
					Value:    left.Value / float64(right.Value),
					Position: position,
				}

			case *parser.ExprFloat:
				data.value = &parser.ExprFloat{
					Value:    left.Value / right.Value,
					Position: position,
				}

			default:
				return &logger.Error{
					Message: "Expect integer or float.",
					Pos:     binary.Right.Pos(),
				}
			}

		default:
			return &logger.Error{
				Message: "Expect integer or float.",
				Pos:     binary.Left.Pos(),
			}
		}
	}

	return nil
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
		data.items = append(data.items, &itemData{
			value:     item,
			evaluated: false,
		})
	}
}

func (evaluator *Evaluator) evaluateArray(array *parser.ExprArray) error {
	evaluator.evaluateArrayIndices(array)

	for _, item := range evaluator.arrayDatas[array].items {
		if err := evaluator.evaluateItem(item); err != nil {
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
		if _, ok := data.items[item.Key.Value]; ok {
			return &logger.Error{
				Message: "Duplicate key in table.",
				Pos:     item.Key.Pos(),
			}
		}

		// Create entry in table.
		data.items[item.Key.Value] = &itemData{
			value:     item.Value,
			evaluated: false,
		}
	}

	// Exit if there is no parent.
	if table.Parent == nil {
		return nil
	}

	// Unwrap parent.
	parentExpr, err := evaluator.unwrap(table.Parent)
	if err != nil {
		return err
	}

	// Make sure the parent is a table.
	parent, ok := parentExpr.(*parser.ExprTable)
	if !ok {
		return &logger.Error{
			Message: "Expect table for parent.",
			Pos:     table.Parent.Pos(),
		}
	}

	// Make sure the parent keys are evaluated.
	if err := evaluator.evaluateTableKeys(parent); err != nil {
		return err
	}

	parentData := evaluator.tableDatas[parent]

	for key, item := range parentData.items {
		// Merge item from parent if key is not already there.
		if _, ok := data.items[key]; !ok {
			data.items[key] = item
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateTable(table *parser.ExprTable) error {
	if err := evaluator.evaluateTableKeys(table); err != nil {
		return err
	}

	for _, item := range evaluator.tableDatas[table].items {
		if err := evaluator.evaluateItem(item); err != nil {
			return err
		}
	}

	return nil
}

func (evaluator *Evaluator) evaluateItem(item *itemData) error {
	// Exit if item is already evaluated.
	if item.evaluated {
		return nil
	}

	// Mark item as evaluated.
	item.evaluated = true

	// Evaluate value.
	if err := evaluator.evaluate(item.value); err != nil {
		return err
	}

	// Unwrap value.
	valueExpr, err := evaluator.unwrap(item.value)
	if err != nil {
		return err
	}

	// Replace value with unwraped value.
	item.value = valueExpr

	return nil
}

func (evaluator *Evaluator) unwrap(expr parser.Expr) (parser.Expr, error) {
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
		if err := evaluator.evaluateRef(current); err != nil {
			return nil, err
		}

		return evaluator.refDatas[current].value, nil

	case *parser.ExprUnary:
		if err := evaluator.evaluateUnary(current); err != nil {
			return nil, err
		}

		return evaluator.unaryDatas[current].value, nil

	case *parser.ExprBinary:
		if err := evaluator.evaluateBinary(current); err != nil {
			return nil, err
		}

		return evaluator.binaryDatas[current].value, nil
	}

	// Expression is invalid if it somehow does not match.
	return nil, &logger.Error{
		Message: "Invalid expression.",
		Pos:     expr.Pos(),
	}
}

func (evaluator *Evaluator) resolve(expr parser.Expr) (any, error) {
	expr, err := evaluator.unwrap(expr)
	if err != nil {
		return nil, err
	}

	switch current := expr.(type) {
	case *parser.ExprStr:
		return current.Value, nil

	case *parser.ExprBool:
		return current.Value, nil

	case *parser.ExprInt:
		return current.Value, nil

	case *parser.ExprFloat:
		return current.Value, nil

	case *parser.ExprArray:
		items := []any{}

		for _, item := range evaluator.arrayDatas[current].items {
			unwrapped, err := evaluator.resolve(item.value)
			if err != nil {
				return nil, err
			}

			items = append(items, unwrapped)
		}

		return items, nil

	case *parser.ExprTable:
		items := map[string]any{}

		for key, item := range evaluator.tableDatas[current].items {
			unwrapped, err := evaluator.resolve(item.value)
			if err != nil {
				return nil, err
			}

			items[key] = unwrapped
		}

		return items, nil
	}

	return nil, &logger.Error{
		Message: "Invalid expression.",
		Pos:     expr.Pos(),
	}
}
