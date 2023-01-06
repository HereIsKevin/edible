from __future__ import annotations

from dataclasses import dataclass, field
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
    _parent: Expr | None = None
    _cache: Expr | None = None
    _evaluated: bool = False

    def __repr__(self) -> str:
        return repr(self._cache)


class UnaryOp(Enum):
    PLUS = auto()
    MINUS = auto()


@dataclass
class Unary:
    op: UnaryOp
    right: Expr

    # Data for evaluation.
    _cache: Expr | None = None
    _evaluated: bool = False

    # def __repr__(self) -> str:
    #     return repr(self._cache)


class BinaryOp(Enum):
    PLUS = auto()
    MINUS = auto()
    STAR = auto()
    SLASH = auto()


@dataclass
class Binary:
    left: Expr
    op: BinaryOp
    right: Expr

    # Data for evaluation.
    _cache: Expr | None = None
    _evaluated: bool = False

    # def __repr__(self) -> str:
    #     return repr(self._cache)


@dataclass
class Array:
    items: list[Expr]

    # Data for evaluation.
    _cache: list[Expr] = field(default_factory=list)

    # def __repr__(self) -> str:
    #     return repr(self._cache)


@dataclass
class TableItem:
    key: Expr
    value: Expr
    parent: Expr | None = None


@dataclass
class Table:
    items: list[TableItem]

    # Data for evaluation.
    _cache: dict[str, Expr | TableItem] = field(default_factory=dict)

    def __repr__(self) -> str:
        return repr(self.items)


Expr: TypeAlias = Str | Bool | Int | Float | Ref | Unary | Binary | Array | Table


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
                return
            case Ref():
                expr._parent = parent
            case Unary(right=right):
                self._resolve(parent, right)
            case Binary(left=left, right=right):
                self._resolve(parent, left)
                self._resolve(parent, right)
            case Array(items):
                for array_item in items:
                    self._resolve(parent, array_item)
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
            case Unary():
                raise NotImplementedError
            case Binary():
                raise NotImplementedError
            case Array():
                raise NotImplementedError
            case Table():
                self._evaluate_table(expr)

    def _evaluate_ref(self, ref: Ref) -> None:
        # check if reference is being evaluated.
        #   if it is, exit.
        #   otherwise, continue and set it the status to being evaluated.
        #
        # use resolved parent or root depending on modifier.
        #
        # for every key in the reference.
        #   evaluate the key recursively.
        #   if the current expression to be indexed is a table.
        #     evaluate each key if possible.
        #     check if the key fits.
        #   if none of the keys fit.
        #     then it is an error.
        #   otherwise update the current expression to the indexed expression.

        if ref._evaluated:
            return

        ref._evaluated = True

        if ref.modifier == RefModifier.ABSOLUTE:
            current = self._expr
        else:
            current = ref._parent

        for key in ref.keys:
            key = self._evaluate_key(key)

            match current:
                case Array():
                    raise NotImplementedError
                case Table(items):
                    found = False

                    for item in items:
                        # item_key = item.key

                        # while isinstance(item_key, Ref):  # do the same for item keys
                        #     self._evaluate(item_key)
                        #     item_key = item_key._cache

                        try:
                            item_key = self._evaluate_key(item.key)
                        except (RuntimeError, TypeError):
                            print("FAILURE", key, item.key, current)
                            continue

                        # maybe check if key is 100% evaluated by checking for nones, nah

                        # print(key.value, type(key.value))
                        # print(item_key.value, type(item_key.value))
                        # print(key.value == item_key.value)

                        if isinstance(item_key, Str) and key.value == item_key.value:
                            print("MATCH", key, item.key, current)
                            current = item.value
                            found = True
                            break

                        print("OK", key, item.key, current)

                    if not found:
                        raise KeyError("Key not found.", key, item_key)

        ref._cache = current

        # ref._cache = current
        # # self._evaluate(ref._cache)

    def _evaluate_key(self, key: Expr) -> Str | Int:
        while not isinstance(key, Str | Int):
            if isinstance(key, Ref):
                self._evaluate(key)

                if key._cache is None:
                    raise RuntimeError("Failed to evaluate reference.")

                key = key._cache
            else:
                raise TypeError("Expected string or integer for key.")

        return key

    # def _evaluate_table_keys(self, table: Table) -> None:
    #     for item in table.items:
    #         self._evaluate_key(item.key)

    def _evaluate_table(self, table: Table) -> None:
        for item in table.items:
            self._evaluate(item.key)
            self._evaluate(item.value)


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
    expr = Table([
        TableItem(Ref([Str("a"), Ref([Str("d")])]), Table([
            TableItem(Str("c"), Str("10")),
            TableItem(Str("z"), Str("c")),
            TableItem(Str("b"), Ref([Str("a"), Ref([Str("10"), Str("z")])])),
        ])),
        TableItem(Str("a"), Table([
            TableItem(Str("c"), Str("10")),
        ])),
        TableItem(Str("d"), Str("c")),
    ])
    # fmt: on

    print(Evaluator(expr).evaluate())
