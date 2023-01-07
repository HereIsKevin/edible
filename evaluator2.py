from __future__ import annotations

from dataclasses import dataclass
from enum import Enum, auto
from typing import TypeAlias


@dataclass
class Str:
    value: str

    def __repr__(self) -> str:
        return repr(self.value)


@dataclass
class Bool:
    value: bool

    def __repr__(self) -> str:
        return repr(self.value)


@dataclass
class Int:
    value: int

    def __repr__(self) -> str:
        return repr(self.value)


@dataclass
class Float:
    value: float

    def __repr__(self) -> str:
        return repr(self.value)


class RefModifier(Enum):
    ABSOLUTE = auto()
    RELATIVE = auto()


@dataclass
class Ref:
    keys: list[Expr]
    modifier: RefModifier = RefModifier.ABSOLUTE

    # Data for evaluation.
    parent: Expr | None = None
    cache: Expr | None = None
    evaluated: bool = False

    def __repr__(self) -> str:
        return repr(self.cache)

@dataclass
class TableItem:
    key: Expr
    value: Expr
    parent: Expr | None = None


@dataclass
class Table:
    items: list[TableItem]

    def __repr__(self) -> str:
        return repr(self.items)


Expr: TypeAlias = Str | Bool | Int | Float | Ref | Table


class Evaluator:
    _expr: Expr

    def __init__(self, expr: Expr) -> None:
        self._expr = expr

    def evaluate(self) -> Expr:
        if isinstance(self._expr, Table):
            parent = self._expr
        else:
            parent = None

        self._resolve(parent, self._expr)
        self._evaluate(self._expr)

        return self._expr

    def _resolve(self, parent: Expr | None, expr: Expr) -> None:
        match expr:
            case Str() | Bool() | Int() | Float():
                pass
            case Ref():
                expr.parent = parent
            case Table(items):
                for table_item in items:
                    self._resolve(expr, table_item.key)
                    self._resolve(expr, table_item.value)

                    if table_item.parent is not None:
                        self._resolve(expr, table_item.parent)

    def _evaluate(self, expr: Expr) -> None:
        match expr:
            case Str() | Bool() | Int() | Float():
                pass
            case Ref():
                self._evaluate_ref(expr)
            case Table():
                self._evaluate_table(expr)

    def _evaluate_ref(self, ref: Ref) -> None:
        if ref.evaluated:
            print("HOT EXIT")
            return

        ref.evaluated = True

        if ref.modifier == RefModifier.ABSOLUTE:
            current: Expr | None = self._expr
        else:
            current = ref.parent

        for key in ref.keys:
            if current is None:
                raise TypeError("NONE!")

            curry = self._unwrap(current)

            key = self._unwrap(key)

            if not isinstance(key, Str):
                raise TypeError("Expect string key.")

            if not isinstance(curry, Table):
                raise TypeError("Expect table.")

            for item in curry.items:
                try:
                    item_key = self._unwrap(item.key)
                except RuntimeError:
                    continue

                if not isinstance(item_key, Str):
                    raise TypeError("Expect string key for item.")

                if key.value == item_key.value:
                    current = item.value
                    break

        ref.cache = current

    def _evaluate_table(self, table: Table) -> None:
        for item in table.items:
            self._evaluate(item.key)
            self._evaluate(item.value)

    def _unwrap(self, expr: Expr) -> Expr:
        while isinstance(expr, Ref):
            self._evaluate(expr)

            if expr.cache is None:
                raise RuntimeError("Failed to evaluate reference.")

            expr = expr.cache

        return expr

if __name__ == "__main__":
    # fmt: off
    # expr = Table([
    #     (Ref([Str("a"), Ref([Str("d")])]), Str("c")),
    #     (Str("a"), Table([
    #         (Str("c"), Str("10")),
    #     ])),
    #     (Str("b"), Ref([Str("a"), Ref([Str("10")])])),
    #     (Str("d"), Str("c"))
    # ])
    # expr = Table([
    #     TableItem(Ref([Str("a"), Ref([Str("d")])]), Table([
    #         TableItem(Str("c"), Str("10")),
    #         TableItem(Str("z"), Str("c")),
    #         TableItem(Str("b"), Ref([Str("a"), Ref([Str("10"), Str("z")])])),
    #     ])),
    #     TableItem(Str("a"), Ref([Str("e")])),
    #     TableItem(Str("e"), Table([
    #         TableItem(Str("c"), Str("10")),
    #     ])),
    #     TableItem(Str("d"), Str("c")),
    # ])
    expr = Table([
        TableItem(Ref([Str("a"), Ref([Str("d")])]), Table([
            TableItem(Str("c"), Str("10")),
            TableItem(Str("z"), Str("c")),
            TableItem(Str("b"), Ref([Str("a"), Ref([Str("10"), Str("z")])])),
        ])),
        TableItem(Str("a"), Ref([Str("e")])),
        TableItem(Str("e"), Table([
            TableItem(Str("c"), Str("10")),
        ])),
        TableItem(Str("d"), Str("c")),
    ])
    # fmt: on

    print(Evaluator(expr).evaluate())

