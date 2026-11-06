package luau

// Ported from src/LuauAST/types/operators.ts.

type BinaryOperator string

const (
	BinaryAdd          BinaryOperator = "+"
	BinarySub          BinaryOperator = "-"
	BinaryMul          BinaryOperator = "*"
	BinaryDiv          BinaryOperator = "/"
	BinaryFloorDiv     BinaryOperator = "//"
	BinaryPow          BinaryOperator = "^"
	BinaryMod          BinaryOperator = "%"
	BinaryConcat       BinaryOperator = ".."
	BinaryLess         BinaryOperator = "<"
	BinaryLessEqual    BinaryOperator = "<="
	BinaryGreater      BinaryOperator = ">"
	BinaryGreaterEqual BinaryOperator = ">="
	BinaryEqual        BinaryOperator = "=="
	BinaryNotEqual     BinaryOperator = "~="
	BinaryAnd          BinaryOperator = "and"
	BinaryOr           BinaryOperator = "or"
)

type UnaryOperator string

const (
	UnaryMinus UnaryOperator = "-"
	UnaryNot   UnaryOperator = "not"
	UnaryLen   UnaryOperator = "#"
)

type AssignmentOperator string

const (
	AssignEqual    AssignmentOperator = "="
	AssignAdd      AssignmentOperator = "+="
	AssignSub      AssignmentOperator = "-="
	AssignMul      AssignmentOperator = "*="
	AssignDiv      AssignmentOperator = "/="
	AssignFloorDiv AssignmentOperator = "//="
	AssignMod      AssignmentOperator = "%="
	AssignPow      AssignmentOperator = "^="
	AssignConcat   AssignmentOperator = "..="
)
