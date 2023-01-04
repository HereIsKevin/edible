package interpreter

import (
	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
)

type Interpreter struct {
	expr   parser.Expr
	logger *logger.Logger

	tables   []*parser.ExprTable
	hadError bool
}

func New(expr parser.Expr, logger *logger.Logger) *Interpreter {
	return &Interpreter{
		expr:   expr,
		logger: logger,

		tables:   []*parser.ExprTable{},
		hadError: false,
	}
}

func (interp *Interpreter) Interpret() parser.Expr {
	interp.interpretExpr(interp.expr)
	return interp.expr
}

func (interp *Interpreter) interpretExpr(expr parser.Expr) {
	switch inner := expr.(type) {
	// Integers, Floats, Booleans, and Strings
	case *parser.ExprStr, *parser.ExprBool, *parser.ExprInt, *parser.ExprFloat:
		// Ignore and move on from those values, they cannot be evaluated more.
		return

	case *parser.ExprRef:
		interp.interpretRef(expr, inner)

	case *parser.ExprUnary:
		interp.interpretUnary(inner)

	case *parser.ExprBinary:
		interp.interpretBinary(inner)

	case *parser.ExprArray:
		interp.interpretArray(inner)

	case *parser.ExprTable:
		interp.interpretTable(inner)
	}
}

func (interp *Interpreter) interpretRef(expr parser.Expr, ref *parser.ExprRef) {
	current := interp.expr
	if ref.Modifier == parser.RefRelative {
		current = interp.lastTable()
	}

	for _, key := range ref.Keys {
		interp.interpretExpr(current)
		if interp.hadError {
			return
		}

		interp.interpretExpr(key)
		if interp.hadError {
			return
		}

		switch key := key.(type) {
		case *parser.ExprStr:
			table, ok := current.(*parser.ExprTable)
			if !ok {
				interp.addError("Only tables have string keys", key)
				return
			}

			current = table.Cache[key.Value]

		case *parser.ExprInt:
			array, ok := current.(*parser.ExprArray)
			if !ok {
				interp.addError("Only arrays have integer keys", key)
			}

			current = array.Items[key.Value]
		}
	}
}

func (interp *Interpreter) interpretUnary(uanry *parser.ExprUnary) {

}

func (interp *Interpreter) interpretBinary(binary *parser.ExprBinary) {

}

func (interp *Interpreter) interpretArray(array *parser.ExprArray) {
	for _, item := range array.Items {
		interp.interpretExpr(item)
		if interp.hadError {
			return
		}
	}
}

func (interp *Interpreter) interpretTable(table *parser.ExprTable) {
	// Temporarily set this to current table for referencing.
	length := interp.saveTable(table)
	defer interp.restoreTables(length)

	// Exit if already interpreted.
	if len(table.Cache) > 0 {
		return
	}

	for _, item := range table.Items {
		// Interpret the key.
		interp.interpretExpr(item.Key)
		if interp.hadError {
			return
		}

		switch key := item.Key.(type) {
		case *parser.ExprStr:
			// Interpret the parent.
			interp.interpretExpr(item.Inherits)
			if interp.hadError {
				return
			}

			// Interpret the value.
			interp.interpretExpr(item.Value)
			if interp.hadError {
				return
			}

			// Check for duplicate keys.
			// _, ok := table.Cache[key.Value]
			// if ok {
			// 	interp.addError("Duplicate key in table", key)
			// 	return
			// }

			// Cache the value.
			table.Cache[key.Value] = item.Value

			// Make sure the value is a table.
			itemTable, ok := item.Value.(*parser.ExprTable)
			if !ok {
				continue
			}

			// Make sure the parent is a table.
			parentTable, ok := item.Inherits.(*parser.ExprTable)
			if !ok {
				continue
			}

			// Merge keys from parent table into value.
			for key, value := range parentTable.Cache {
				_, ok := itemTable.Cache[key]
				if ok {
					continue
				}

				itemTable.Cache[key] = value
			}
		default:
			interp.addError("Expect string for table key", key)
			return
		}
	}
}

func (interp *Interpreter) saveTable(table *parser.ExprTable) int {
	length := len(interp.tables)
	interp.tables = append(interp.tables, table)

	return length
}

func (interp *Interpreter) lastTable() *parser.ExprTable {
	if len(interp.tables) == 0 {
		return nil
	}

	return interp.tables[len(interp.tables)-1]
}

func (interp *Interpreter) restoreTables(length int) {
	if len(interp.tables) > length {
		interp.tables = interp.tables[:length]
	}
}

func (interp *Interpreter) addError(message string, expr parser.Expr) {
	interp.logger.Add(message, expr.Span())
	interp.hadError = true
}
