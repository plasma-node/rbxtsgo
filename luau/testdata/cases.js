"use strict";
// Hand-written golden cases. Every case builds an AST with the real npm
// @roblox-ts/luau-ast 2.0.0 and names how to render it:
//
//   renderAST                 luau.renderAST(list)
//   renderStatements          new RenderState, solveTempIds, renderStatements(list)
//   render                    new RenderState, solveTempIds, render(node)
//   renderUnsolved            new RenderState, render(node)         (temp ids use the fallback names)
//   renderStatementsUnsolved  new RenderState, renderStatements(list)

const luau = require("@roblox-ts/luau-ast").default;
const K = luau.SyntaxKind;
const c = luau.create;

const cases = [];
function add(name, mode, build) {
	cases.push({ name, mode, build });
}
const ast = (name, build) => add(name, "renderAST", build);

// builders ------------------------------------------------------------------
const L = (...xs) => luau.list.make(...xs);
const id = luau.id;
const tmp = luau.tempId;
const num = value => c(K.NumberLiteral, { value });
const str = luau.string;
const nil = luau.nil;
const varargs = () => c(K.VarArgsLiteral, {});
const prop = luau.property;
const call = (e, ...args) => luau.call(e, args);
const mcall = (e, name, ...args) => c(K.MethodCallExpression, { name, expression: e, args: L(...args) });
const paren = e => c(K.ParenthesizedExpression, { expression: e });
const bin = luau.binary;
const un = luau.unary;
const index = (e, i) => c(K.ComputedIndexExpression, { expression: e, index: i });
const ifx = (condition, expression, alternative) => c(K.IfExpression, { condition, expression, alternative });
const fn = (params, body, hasDotDotDot = false) =>
	c(K.FunctionExpression, { parameters: L(...params), hasDotDotDot, statements: L(...body) });
const interp = (...parts) => c(K.InterpolatedString, { parts: L(...parts) });
const part = text => c(K.InterpolatedStringPart, { text });
const local = (left, right) => c(K.VariableDeclaration, { left, right });
const assign = (left, operator, right) => c(K.Assignment, { left, operator, right });
const set = (left, right) => assign(left, "=", right);
const stmt = e => c(K.CallStatement, { expression: e });
const ret = expression => c(K.ReturnStatement, { expression });
const brk = () => c(K.BreakStatement, {});
const cont = () => c(K.ContinueStatement, {});
const comment = luau.comment;
const doBlock = (...body) => c(K.DoStatement, { statements: L(...body) });
const whileS = (condition, ...body) => c(K.WhileStatement, { condition, statements: L(...body) });
const repeatS = (condition, ...body) => c(K.RepeatStatement, { condition, statements: L(...body) });
const ifS = (condition, statements, elseBody = L()) => c(K.IfStatement, { condition, statements: L(...statements), elseBody });
const forS = (ids, expression, ...body) => c(K.ForStatement, { ids: L(...ids), expression, statements: L(...body) });
const nfor = (id_, start, end, step, ...body) =>
	c(K.NumericForStatement, { id: id_, start, end, step, statements: L(...body) });
const fnDecl = (name, params, body, localize = false, hasDotDotDot = false) =>
	c(K.FunctionDeclaration, { localize, name, parameters: L(...params), hasDotDotDot, statements: L(...body) });
const methodDecl = (expression, name, params, body, hasDotDotDot = false) =>
	c(K.MethodDeclaration, { expression, name, parameters: L(...params), hasDotDotDot, statements: L(...body) });
const f = (...args) => stmt(call(id("f"), ...args)); // f(args)
const x = () => id("x");

const BINARY_OPS = ["+", "-", "*", "/", "//", "^", "%", "..", "<", "<=", ">", ">=", "==", "~=", "and", "or"];
const UNARY_OPS = ["-", "not", "#"];
const ASSIGN_OPS = ["=", "+=", "-=", "*=", "/=", "//=", "%=", "^=", "..="];

// literals --------------------------------------------------------------------
ast("literal-keywords", () =>
	L(
		set(x(), nil()),
		set(x(), luau.bool(true)),
		set(x(), luau.bool(false)),
		fnDecl(id("g"), [], [local(id("a"), varargs())]),
	),
);

const VALID_NUMBERS = [
	"0", "1", "123", "1.5", "0.5", ".5", "5.", "1.e3", "1e3", "1E3", "1e+3", "1E-3", "1_000", "1__0", "0_1", "1_000.5_e1_0",
	"1e_5", "1e+_5", "0x1F", "0XfF", "0x_1_", "0_x1", "0b101", "0B1", "0b_1_", "00012", "0.0000001", "1e1000",
	"123456789012345678901234567890", "0xFFFFFFFFFFFFFFFFFFFF", "9007199254740993", "3.14159265358979323846",
];
VALID_NUMBERS.forEach((v, i) => ast(`number-valid-${i}`, () => L(set(x(), num(v)))));

// Not valid Luau literals: rendered as String(Number(value.replace(/_/g, ""))).
const FALLBACK_NUMBERS = [
	"", " ", "0o17", "0O777", "0o1_7", "0o8", "0b102", "0x", "0xg", "Infinity", "+Infinity", "-Infinity", "infinity", "NaN",
	"-5", "+5", "+0", "-0", "+.5", "+5.", "-.5e1", "1e", "1e+", "e5", ".", "-", "+", "- 5", "+0x10", "-0x10", "1 2", " 12 ",
	"\t7\n", " 7", " 7", "7​", "﻿7", "+1e21", "+1e22", "+1e-7", "+1e-6", "+0.000001", "+0.0000001",
	"+0.1", "+123456789012345678901", "+1234567890123456789012", "+9007199254740993", "-1.5e300", "+5e-324",
	"+1.7976931348623157e308", "+1.7976931348623159e308", "-1e400", "+1e400", "0o7777777777777777777777",
	"0x1fffffffffffff", "0x20000000000001", "0xfffffffffffff800", "0xfffffffffffffc00", "0b" + "1".repeat(60), "+1_0",
	"+1_000.5", "١٢", "1n", "0.1.2", "+00012", "+1e0", "+10e-1", "+4.35", "+0.3", "+2e-7", "+123e-20", "+1.5e+25",
	"+100000000000000000000", "+1000000000000000000000", "+999999999999999900000", "0x1p3",
];
FALLBACK_NUMBERS.forEach((v, i) => ast(`number-fallback-${i}`, () => L(set(x(), num(v)))));

// numeric for: a step that is the number 1 is omitted
const STEPS = ["1", "1.0", "1e0", "01", "0x1", "0b1", "0o1", "1_0", "+1", " 1 ", "1.00", "0.1e1", "2", "0", "-1", "NaN", "1.0000000000000001"];
STEPS.forEach((v, i) => ast(`for-step-literal-${i}`, () => L(nfor(id("i"), num("1"), num("10"), num(v), f(id("i"))))));
ast("for-step-kinds", () =>
	L(
		nfor(id("i"), num("1"), num("10"), undefined, f(id("i"))),
		nfor(id("i"), num("1"), num("10"), id("step"), f(id("i"))),
		nfor(id("i"), num("10"), num("1"), un("-", num("1")), f(id("i"))),
		nfor(id("i"), num("10"), num("1"), luau.number(-1), f(id("i"))),
		nfor(id("i"), num("1"), num("10"), call(id("n")), f(id("i"))),
		nfor(tmp("i"), id("a"), bin(id("b"), "+", num("1")), undefined),
	),
);

// strings ---------------------------------------------------------------------
const STRINGS = [
	"", "abc", 'a"b', "a'b", `a"b'c`, "a\nb", "\n", `a"b\nc`, "a]]b", "a]=]b", "a]==]b", "a]]b]=]c]==]d", "ends with ]",
	"ends with ]=", "ends with ]==", "]", "]=", "[[", "]]", "]=]", "a\rb", "\r", 'a\r"b', "back\\slash", 'q"\\', "tab\there",
	"é😀 ünï", " ", "x\ny]]", "x\n]=", "\n]", "a\n]]b]=]c]==]d", "\n]]]=]]==]", "\n]]]=]]==]]===]", "\n]]]=]x]=", "\n]]]=]]==]x]==", "\n]=]]==]", "end", "valid_id", "9bad", "with space", "a.b",
];
STRINGS.forEach((v, i) => {
	ast(`string-plain-${i}`, () => L(f(str(v))));
	ast(`string-key-${i}`, () =>
		L(
			set(index(id("t"), str(v)), str(v)),
			local(id("m"), luau.map([[str(v), str(v)]])),
			local(id("s"), luau.set([str(v)])),
			local(id("a"), luau.array([str(v), str(v)])),
			local(id("p"), c(K.MixedTable, { fields: L(str(v), c(K.MapField, { index: str(v), value: str(v) })) })),
		),
	);
});

// identifiers and properties ----------------------------------------------------
const NAMES = ["a", "_", "_a", "a1", "A_b", "end", "and", "nil", "function", "self", "9a", "a-b", "a b", "", "é", "a.b", "type", "continue", "goto"];
NAMES.forEach((v, i) => {
	ast(`property-${i}`, () =>
		L(set(prop(id("o"), v), prop(prop(id("o"), v), v)), set(x(), c(K.ComputedIndexExpression, { expression: id("o"), index: str(v) }))),
	);
	ast(`map-key-${i}`, () => L(local(id("m"), luau.map([[str(v), num("1")], [num("2"), str(v)]]))));
});

// error cases: invalid identifiers and node kinds
["a-b", "end", "", "9a", "a b", "é"].forEach((v, i) => ast(`error-identifier-${i}`, () => L(set(id(v), num("1")))));
ast("error-method-name-invalid", () => L(stmt(mcall(id("o"), "a-b"))));
ast("error-method-name-keyword", () => L(stmt(mcall(id("o"), "end"))));
ast("error-none-statement", () => L(local(id("a"), luau.none())));
ast("error-statement-after-return", () => L(ret(num("1")), f()));
ast("error-statement-after-break", () => L(whileS(luau.bool(true), brk(), f())));
ast("error-statement-after-continue", () => L(whileS(luau.bool(true), cont(), f())));
ast("ok-comment-after-return", () => L(ret(num("1")), comment("c"), comment("d\ne")));
ast("error-local-function-property", () => L(fnDecl(prop(id("a"), "b"), [], [], true)));
ast("error-empty-assignment-left", () => L(assign(L(), "=", num("1"))));
ast("error-empty-assignment-right", () => L(assign(id("a"), "=", L())));
ast("error-empty-local-left", () => L(local(L(), num("1"))));
ast("error-empty-local-right", () => L(local(id("a"), L())));
ast("ok-empty-return-list", () => L(ret(L())));
ast("error-temp-invalid-name", () => L(local(tmp("a b"), num("1"))));
ast("error-temp-keyword-name", () => L(local(tmp("end"), num("1"))));
add("error-render-none", "render", () => luau.none());
add("error-render-none-unsolved", "renderUnsolved", () => luau.none());

// operators and parentheses ---------------------------------------------------------
BINARY_OPS.forEach((op1, i) =>
	ast(`binary-matrix-${i}`, () =>
		L(
			...BINARY_OPS.flatMap(op2 => [
				set(x(), bin(bin(id("a"), op1, id("b")), op2, id("c"))),
				set(x(), bin(id("a"), op2, bin(id("b"), op1, id("c")))),
			]),
		),
	),
);
BINARY_OPS.forEach((op, i) =>
	ast(`unary-binary-${i}`, () =>
		L(
			...UNARY_OPS.flatMap(u => [
				set(x(), un(u, bin(id("a"), op, id("b")))),
				set(x(), bin(un(u, id("a")), op, id("b"))),
				set(x(), bin(id("a"), op, un(u, id("b")))),
				set(x(), un(u, bin(un(u, id("a")), op, un(u, id("b"))))),
			]),
		),
	),
);
ast("unary-nested", () =>
	L(
		...UNARY_OPS.flatMap(u1 => UNARY_OPS.map(u2 => set(x(), un(u1, un(u2, id("a")))))),
		set(x(), un("-", un("-", un("-", id("a"))))),
		set(x(), un("-", luau.number(-5))),
		set(x(), un("-", num("5"))),
		set(x(), un("not", un("not", un("not", id("a"))))),
		set(x(), un("-", paren(un("-", id("a"))))),
		set(x(), un("#", call(id("f")))),
		set(x(), un("-", prop(id("a"), "b"))),
	),
);
ast("number-constructor", () =>
	L(
		set(x(), luau.number(0)),
		set(x(), luau.number(5)),
		set(x(), luau.number(-5)),
		set(x(), luau.number(1.5)),
		set(x(), luau.number(-0.25)),
		set(x(), luau.number(1e21)),
		set(x(), luau.number(1e-7)),
		set(x(), luau.number(123456789012)),
		set(x(), luau.number(Infinity)),
		set(x(), luau.number(-Infinity)),
		set(x(), bin(luau.number(-1), "^", luau.number(-2))),
		set(x(), bin(luau.number(-1), "-", luau.number(-2))),
	),
);
ast("ifexpr-precedence", () => {
	const mk = () => ifx(id("c"), id("a"), id("b"));
	return L(
		set(x(), mk()),
		set(x(), ifx(id("c1"), id("a1"), ifx(id("c2"), id("a2"), id("b")))),
		set(x(), ifx(id("c1"), id("a1"), ifx(id("c2"), id("a2"), ifx(id("c3"), id("a3"), id("b"))))),
		set(x(), ifx(ifx(id("c"), id("a"), id("b")), id("d"), id("e"))),
		set(x(), ifx(id("c"), ifx(id("c2"), id("a"), id("b")), id("e"))),
		set(x(), ifx(id("c"), id("a"), paren(ifx(id("c2"), id("a2"), id("b"))))),
		...BINARY_OPS.flatMap(op => [
			set(x(), bin(mk(), op, id("z"))),
			set(x(), bin(id("z"), op, mk())),
			set(x(), ifx(bin(id("a"), op, id("b")), bin(id("c"), op, id("d")), bin(id("e"), op, id("f")))),
		]),
		...UNARY_OPS.map(u => set(x(), un(u, mk()))),
		set(x(), mk()),
		set(x(), call(id("f"), mk(), ifx(id("c"), id("a"), id("b")))),
		set(x(), index(id("t"), mk())),
	);
});
ast("parenthesized", () =>
	L(
		set(x(), paren(id("a"))),
		set(x(), paren(paren(id("a")))),
		set(x(), paren(paren(paren(num("1"))))),
		set(x(), paren(str("s"))),
		set(x(), paren(nil())),
		set(x(), paren(luau.bool(true))),
		set(x(), paren(call(id("f")))),
		set(x(), paren(paren(call(id("f"))))),
		set(x(), paren(prop(id("a"), "b"))),
		set(x(), paren(bin(id("a"), "+", id("b")))),
		set(x(), paren(paren(bin(id("a"), "+", id("b"))))),
		set(x(), paren(varargs())),
		set(x(), paren(tmp("t"))),
		set(x(), paren(luau.array([]))),
		set(x(), bin(paren(bin(id("a"), "+", id("b"))), "*", id("c"))),
		set(x(), paren(un("-", id("a")))),
		set(x(), paren(paren(paren(id("a"))))),
		set(x(), paren(paren(paren(bin(id("a"), "or", id("b")))))),
	),
);

// calls, properties, indexing ------------------------------------------------------------
ast("calls", () =>
	L(
		f(),
		f(num("1")),
		f(num("1"), str("a"), nil()),
		stmt(mcall(id("o"), "m")),
		stmt(mcall(id("o"), "m", num("1"), id("b"))),
		stmt(mcall(mcall(id("o"), "a"), "b")),
		stmt(call(call(id("f")))),
		stmt(call(prop(prop(id("a"), "b"), "c"), id("d"))),
		stmt(call(index(id("a"), num("1")))),
		stmt(call(paren(id("f")))),
		stmt(mcall(paren(id("o")), "m")),
		set(x(), call(id("f"), call(id("g"), call(id("h"))))),
		set(x(), index(index(id("a"), id("b")), index(id("c"), str("d")))),
		set(x(), index(call(id("f")), str("valid"))),
		set(x(), index(id("a"), str("end"))),
		set(x(), index(id("a"), str("not valid"))),
		set(x(), index(id("a"), str(""))),
		set(x(), index(id("a"), un("-", num("1")))),
		set(x(), prop(call(id("f")), "end")),
		set(x(), prop(mcall(id("o"), "m"), "bad name")),
		set(x(), prop(id("a"), 'quote"d')),
	),
);

// tables ---------------------------------------------------------------------------------
ast("tables", () =>
	L(
		set(x(), luau.array()),
		set(x(), luau.array([num("1")])),
		set(x(), luau.array([num("1"), str("a"), id("b"), luau.array([]), luau.array([num("2")])])),
		set(x(), luau.map()),
		set(x(), luau.map([[str("a"), num("1")]])),
		set(x(), luau.map([[str("a"), num("1")], [num("2"), luau.map([[str("b"), luau.map()]])], [id("k"), id("v")], [str("end"), nil()]])),
		set(x(), luau.set()),
		set(x(), luau.set([str("a"), num("1"), id("b")])),
		set(x(), luau.set([luau.set([id("a")])])),
		set(x(), luau.mixedTable()),
		set(x(), luau.mixedTable([id("a"), [str("k"), id("b")], [num("1"), id("c")], luau.array([id("d")])])),
		set(x(), luau.mixedTable([luau.map([[str("a"), luau.mixedTable([id("b")])]])])),
		f(luau.map([[str("a"), num("1")]]), luau.array([num("1")])),
		set(x(), luau.array([fn([id("a")], [ret(id("a"))]), fn([], [])])),
		set(x(), luau.map([[str("fn"), fn([], [f(), ret(luau.map([[str("a"), num("1")]]))])]])),
		set(x(), luau.map([[tmp("k"), tmp("v")], [tmp("k2"), str("v")]])),
	),
);

// interpolated strings ---------------------------------------------------------------------------
const PART_TEXTS = [
	"", "plain", "a{b}c", "{}", "{{}}", "\\u{41}", "\\u{1F600}x{y}", "\\u{}", "\\u{zz}", "\\u{41", "\\u41", "\\{", "\\\\{",
	"\\", "a\nb", "a\r\nb", "a\rb", "\r\r\n\n", "\n\n", "{\\u{1f}}\n", "\\u{ABCdef012}{", "\\\\u{41}", "\\u{41}\\u{42}}", "`",
	"é{😀}", "\\U{41}", "u{41}", "\\u{4 1}",
];
PART_TEXTS.forEach((t, i) => ast(`interp-part-${i}`, () => L(set(x(), interp(part(t))))));
ast("interp-mixed", () =>
	L(
		set(x(), interp()),
		set(x(), interp(part("a "), id("b"), part(" c"))),
		set(x(), interp(id("a"), id("b"))),
		set(x(), interp(part("t: "), luau.array([id("a")]), part(" "), luau.map(), part(" "), luau.set(), part(" "), luau.mixedTable())),
		set(x(), interp(call(id("f"), str("a")), part("!"), bin(id("a"), "+", id("b")))),
		set(x(), interp(interp(part("in"), id("x")), part("out"))),
		set(x(), interp(fn([], [ret(num("1"))]))),
		set(x(), interp(str("a\nb"), str("a'\"b"))),
		set(x(), interp(tmp("t"), tmp("t"))),
	),
);

// functions --------------------------------------------------------------------------------------
ast("functions", () =>
	L(
		fnDecl(id("a"), [], []),
		fnDecl(id("a"), [id("p")], [ret(id("p"))], true),
		fnDecl(id("a"), [id("p"), id("q")], [f()], false),
		fnDecl(id("a"), [], [], true, true),
		fnDecl(id("a"), [id("p")], [], false, true),
		fnDecl(id("a"), [id("p"), id("q")], [f(varargs())], true, true),
		fnDecl(prop(id("a"), "b"), [], [f()]),
		fnDecl(prop(prop(id("a"), "b"), "c"), [id("p")], [f()], false),
		fnDecl(tmp("fn"), [], [f()], true),
		fnDecl(tmp("fn"), [tmp("p")], [ret(tmp("p"))], false, true),
		methodDecl(id("A"), "m", [id("self")], [f()]),
		methodDecl(prop(id("A"), "B"), "m", [], [], true),
		methodDecl(id("A"), "end", [], []),
		methodDecl(paren(id("A")), "m", [id("p")], [ret(id("p"))], true),
		set(x(), fn([], [])),
		set(x(), fn([id("a")], [])),
		set(x(), fn([id("a"), id("b")], [], true)),
		set(x(), fn([], [], true)),
		set(x(), fn([], [f()])),
		set(x(), fn([id("a")], [local(id("b"), id("a")), ret(id("b"))], false)),
		f(fn([], [f(fn([], [f(fn([], [ret(nil())]))]))])),
		local(id("a"), fn([], [ret(fn([], [ret(luau.map([[str("a"), fn([], [])]]))]))])),
	),
);

// control flow --------------------------------------------------------------------------------------
ast("control-flow", () =>
	L(
		doBlock(),
		doBlock(f()),
		doBlock(doBlock(doBlock(f()))),
		whileS(luau.bool(true)),
		whileS(id("a"), f(), brk()),
		whileS(bin(id("a"), "and", id("b")), cont()),
		repeatS(id("a")),
		repeatS(bin(id("a"), ">", num("1")), f(), cont()),
		forS([], id("t"), f()),
		forS([id("k")], id("t")),
		forS([id("k"), id("v")], call(id("pairs"), id("t")), f(id("k"), id("v"))),
		forS([tmp("k"), tmp("v")], call(id("pairs"), id("t")), f(tmp("k"), tmp("v"))),
		nfor(id("i"), num("1"), num("10"), undefined, f(id("i"))),
		nfor(tmp("i"), num("1"), id("n"), num("2"), f(tmp("i"))),
		ret(num("0")),
	),
);
ast("control-flow-return", () => L(ret(num("1"))));
ast("control-flow-return-list", () => L(ret(L(num("1"), id("a"), call(id("f"))))));
ast("control-flow-return-varargs", () => L(ret(varargs())));
ast("if-statements", () =>
	L(
		ifS(id("a"), []),
		ifS(id("a"), [f()]),
		ifS(id("a"), [f()], L(f(id("e")))),
		ifS(id("a"), [f()], L(f(id("e")), f(id("e2")))),
		ifS(id("a"), [f()], ifS(id("b"), [f(id("b"))])),
		ifS(id("a"), [f()], ifS(id("b"), [f(id("b"))], L(f(id("e"))))),
		ifS(id("a"), [f()], ifS(id("b"), [f(id("b"))], ifS(id("c"), [f(id("c"))], ifS(id("d"), [f(id("d"))], L(f(id("e"))))))),
		ifS(id("a"), [], ifS(id("b"), [], L())),
		ifS(id("a"), [ifS(id("b"), [f()], L(f()))], L(ifS(id("c"), [f()]))),
		ifS(bin(id("a"), "and", id("b")), [ret(id("c"))], L(brk())),
	),
);

// assignments and declarations --------------------------------------------------------------------------
ast("assignments", () =>
	L(
		...ASSIGN_OPS.map(op => assign(id("a"), op, num("1"))),
		assign(L(id("a"), id("b")), "=", L(num("1"), num("2"))),
		assign(L(id("a")), "=", L(id("b"))),
		assign(L(id("a"), prop(id("b"), "c"), index(id("d"), num("1"))), "=", call(id("f"))),
		assign(prop(id("a"), "b"), "+=", L(num("1"))),
		assign(index(id("a"), id("b")), "..=", str("s")),
		assign(tmp("t"), "=", tmp("t")),
		assign(L(tmp("a"), tmp("b")), "=", L(tmp("b"), tmp("a"))),
	),
);
ast("declarations", () =>
	L(
		local(id("a"), undefined),
		local(id("a"), num("1")),
		local(L(id("a"), id("b")), undefined),
		local(L(id("a"), id("b")), L(num("1"), num("2"))),
		local(L(id("a")), call(id("f"))),
		local(L(id("a"), id("b")), call(id("f"))),
		local(id("a"), L(num("1"))),
		local(tmp("t"), undefined),
		local(L(tmp("a"), id("b"), tmp("c")), L(num("1"), num("2"), num("3"))),
	),
);

// comments ----------------------------------------------------------------------------------------------------
const COMMENTS = [
	"", " hi", "a", "a\nb", "\n", "a\n", "\na", "a\n\nb", "x]]y\nz", "x]=]y\nz", "x]==]\n", "x]\n", "x]=\ny", "x]=\n", "a\rb",
	"]]\n]=]", "]]\n]=]\n]==]", "]]\n]=]x]=", "]]\n]=]]==]x]==", "a\r\nb", "multi\nline\n  indented\n\ttabbed", "-- nested", "--[[ x ]]", "é\n😀",
];
COMMENTS.forEach((t, i) =>
	ast(`comment-${i}`, () =>
		L(
			comment(t),
			doBlock(comment(t), f(), comment(t)),
			fnDecl(id("g"), [], [comment(t), ret(nil())]),
		),
	),
);

// semicolons -----------------------------------------------------------------------------------------------------
// A statement that can end in an indexable expression, followed by one starting with "(".
const ENDERS = {
	call: () => stmt(call(id("f"))),
	mcall: () => stmt(mcall(id("o"), "m")),
	"local-id": () => local(id("a"), id("b")),
	"local-none": () => local(id("a"), undefined),
	"local-nones": () => local(L(id("a"), id("b")), undefined),
	"local-list-tail": () => local(L(id("a"), id("b")), L(num("1"), id("c"))),
	"local-list-tail-number": () => local(L(id("a"), id("b")), L(id("c"), num("1"))),
	"local-prop": () => local(id("a"), prop(id("b"), "c")),
	"local-index": () => local(id("a"), index(id("b"), num("1"))),
	"local-paren": () => local(id("a"), paren(id("b"))),
	"local-number": () => local(id("a"), num("1")),
	"local-string": () => local(id("a"), str("s")),
	"local-nil": () => local(id("a"), nil()),
	"local-varargs": () => local(id("a"), varargs()),
	"local-binary-id": () => local(id("a"), bin(num("1"), "+", id("b"))),
	"local-binary-number": () => local(id("a"), bin(id("b"), "+", num("1"))),
	"local-unary-id": () => local(id("a"), un("-", id("b"))),
	"local-unary-number": () => local(id("a"), un("-", num("1"))),
	"local-if-id": () => local(id("a"), ifx(id("c"), num("1"), id("b"))),
	"local-if-number": () => local(id("a"), ifx(id("c"), id("b"), num("1"))),
	"local-if-nested": () => local(id("a"), ifx(id("c"), num("1"), ifx(id("d"), num("2"), id("b")))),
	"local-table": () => local(id("a"), luau.array([id("b")])),
	"local-fn": () => local(id("a"), fn([], [])),
	"local-tmp": () => local(tmp("t"), tmp("u")),
	"assign-id": () => set(id("a"), id("b")),
	"assign-number": () => set(id("a"), num("1")),
	"assign-list": () => set(L(id("a"), id("b")), L(num("1"), call(id("f")))),
	"assign-compound": () => assign(id("a"), "+=", id("b")),
	"do": () => doBlock(),
	"return-none": () => comment("c"),
};
const STARTERS = {
	"call-paren": () => stmt(call(paren(id("f")))),
	"call-paren-nested": () => stmt(call(call(paren(id("f"))))),
	"call-paren-prop": () => stmt(call(prop(paren(id("a")), "b"))),
	"call-paren-index": () => stmt(call(index(paren(id("a")), num("1")))),
	"mcall-paren": () => stmt(mcall(paren(id("o")), "m")),
	"mcall-paren-prop": () => stmt(mcall(prop(paren(id("o")), "p"), "m")),
	"assign-paren-prop": () => set(prop(paren(id("a")), "b"), num("1")),
	"assign-paren-index": () => set(index(paren(id("a")), num("1")), num("1")),
	"assign-paren-list": () => set(L(prop(paren(id("a")), "b"), id("c")), L(num("1"), num("2"))),
	"assign-paren-list-second": () => set(L(id("c"), prop(paren(id("a")), "b")), L(num("1"), num("2"))),
	"call-plain": () => stmt(call(id("g"))),
	"call-plain-arg-paren": () => stmt(call(id("g"), paren(id("a")))),
	"local-paren": () => local(id("a"), paren(id("b"))),
	"assign-plain": () => set(id("a"), paren(id("b"))),
	"do": () => doBlock(),
	"nonstmt-comment": () => comment("c"),
	"return-paren": () => ret(paren(id("a"))),
};
for (const [en, enderFn] of Object.entries(ENDERS)) {
	ast(`semicolon-${en}`, () => L(...Object.values(STARTERS).flatMap(starter => [enderFn(), starter()])));
	ast(`semicolon-comments-${en}`, () =>
		L(
			...Object.values(STARTERS).flatMap(starter => [enderFn(), comment("c1"), comment("c2\nc3"), starter()]),
		),
	);
	ast(`semicolon-last-${en}`, () => L(enderFn()));
	ast(`semicolon-then-comment-end-${en}`, () => L(enderFn(), comment("c")));
}
ast("semicolon-in-blocks", () =>
	L(
		fnDecl(id("g"), [], [local(id("a"), id("b")), stmt(call(paren(id("f"))))]),
		doBlock(stmt(call(id("a"))), set(prop(paren(id("a")), "b"), num("1"))),
		ifS(id("a"), [stmt(call(id("a")))], L(stmt(call(paren(id("b")))))),
		stmt(call(id("a"))),
		fnDecl(id("g"), [], [stmt(call(id("a")))]),
		stmt(call(paren(id("b")))),
	),
);

// temporary identifiers --------------------------------------------------------------------------------------------------
ast("temp-basic", () => {
	const a = tmp("a"), b = tmp("a"), c = tmp("");
	return L(local(a, num("1")), local(b, num("2")), local(c, num("3")), f(a, b, c));
});
ast("temp-same-id-reused", () => {
	const a = tmp("a");
	return L(local(a, num("1")), f(a), f(a, a), set(x(), bin(a, "+", a)));
});
ast("temp-name-collisions", () => {
	// _x declared, _x_1 declared, temps named x
	const t1 = tmp("x"), t2 = tmp("x"), t3 = tmp("x");
	return L(local(id("_x"), num("1")), local(id("_x_1"), num("2")), local(t1, num("3")), local(t2, num("4")), local(t3, num("5")), f(t1, t2, t3));
});
ast("temp-unnamed-collisions", () => {
	const t1 = tmp(""), t2 = tmp(""), t3 = tmp("");
	return L(local(id("_"), num("1")), local(id("_1"), num("1")), local(t1, num("3")), local(t2, num("4")), local(t3, num("5")), f(t1, t2, t3));
});
ast("temp-declared-after", () => {
	const t1 = tmp("v");
	return L(local(t1, num("1")), local(id("_v"), num("2")), local(id("_v_1"), num("2")), f(t1));
});
ast("temp-function-scopes", () => {
	const o1 = tmp("v"), i1 = tmp("v"), i2 = tmp("v"), o2 = tmp("v");
	return L(
		local(o1, num("1")),
		fnDecl(id("g"), [id("_v")], [local(i1, num("2")), local(i2, num("3")), f(i1, i2, o1)]),
		local(o2, num("4")),
		f(o1, o2),
	);
});
ast("temp-function-expression-scopes", () => {
	const a = tmp("p"), b = tmp("p"), c = tmp("p");
	return L(
		local(a, fn([id("_p")], [local(b, num("1")), f(b)])),
		local(c, fn([tmp("p")], [f()])),
		f(a, c),
	);
});
ast("temp-method-scopes", () => {
	const a = tmp("m"), b = tmp("m");
	return L(local(a, num("1")), methodDecl(id("A"), "m", [id("_m"), id("self")], [local(b, num("2")), f(b, a)]));
});
ast("temp-sibling-scopes", () => {
	const a = tmp("s"), b = tmp("s"), c = tmp("s");
	return L(
		doBlock(local(a, num("1")), f(a)),
		doBlock(local(b, num("2")), f(b)),
		whileS(id("w"), local(c, num("3")), f(c)),
	);
});
ast("temp-do-scope-nested", () => {
	const a = tmp("n"), b = tmp("n"), c = tmp("n"), d = tmp("n");
	return L(
		local(a, num("1")),
		doBlock(local(b, num("2")), doBlock(local(c, num("3")), f(a, b, c))),
		local(d, num("4")),
		f(d),
	);
});
ast("temp-if-else-scopes", () => {
	const a = tmp("q"), b = tmp("q"), c = tmp("q"), d = tmp("q"), e = tmp("q");
	return L(
		local(a, num("0")),
		ifS(id("c1"), [local(b, num("1")), f(b)], ifS(id("c2"), [local(c, num("2")), f(c)], L(local(d, num("3")), f(d)))),
		f(a, e),
	);
});
ast("temp-else-list-scope", () => {
	const a = tmp("q"), b = tmp("q"), c = tmp("q");
	return L(ifS(id("c1"), [local(a, num("1"))], L(local(b, num("2")), local(c, num("3")), f(b, c))), f(a));
});
ast("temp-else-declared", () => {
	const a = tmp("q"), b = tmp("q");
	return L(
		ifS(id("c1"), [local(a, num("1")), f(a)], L(local(id("_q"), num("2")), f(b))),
	);
});
ast("temp-for-ids-not-registered", () => {
	const t = tmp("k"), t2 = tmp("k");
	return L(
		forS([id("_k")], id("tbl"), local(t, num("1")), f(t)),
		nfor(id("_k"), num("1"), num("2"), undefined, local(t2, num("1")), f(t2)),
	);
});
ast("temp-for-loops", () => {
	const k = tmp("k"), v = tmp("v"), i = tmp("i"), inner = tmp("i");
	return L(
		forS([k, v], call(id("pairs"), id("t")), f(k, v), local(inner, k)),
		nfor(i, num("1"), num("3"), undefined, f(i), local(tmp("i"), i)),
	);
});
ast("temp-for-expression-scope", () => {
	// the expression of a for statement is inside the for scope
	const a = tmp("e"), b = tmp("e");
	return L(local(a, num("1")), forS([id("k")], bin(b, "+", a), f()));
});
ast("temp-shadow-params", () => {
	const a = tmp("a"), b = tmp("b");
	return L(fnDecl(id("g"), [id("_a"), id("_b"), id("_b_1")], [local(a, num("1")), local(b, num("2")), f(a, b)], true, true));
});
ast("temp-multi-declaration", () => {
	const a = tmp("a"), b = tmp("a");
	return L(local(L(id("_a"), a, b), L(num("1"), num("2"), num("3"))), f(a, b));
});
ast("temp-inside-expressions", () => {
	const a = tmp("x"), b = tmp("x"), c = tmp("x");
	return L(
		set(prop(a, "p"), index(b, c)),
		f(call(a, b), mcall(c, "m", a), paren(b), un("-", c), ifx(a, b, c)),
	);
});
ast("temp-many", () => {
	const ts = [];
	for (let i = 0; i < 12; i++) ts.push(tmp("t"));
	return L(...ts.map(t => local(t, num("1"))), f(...ts));
});
ast("temp-many-unnamed", () => {
	const ts = [];
	for (let i = 0; i < 12; i++) ts.push(tmp(""));
	return L(...ts.map(t => local(t, num("1"))), f(...ts));
});
ast("temp-names-with-underscore", () => {
	const a = tmp("a_1"), b = tmp("a"), c = tmp("a"), d = tmp("_");
	return L(local(a, num("1")), local(b, num("2")), local(c, num("3")), local(d, num("4")), f(a, b, c, d));
});
ast("temp-collision-chain", () => {
	// names that interact: _a, _a_1, then temps named "a" and "a_1"
	const a1 = tmp("a"), a2 = tmp("a"), b1 = tmp("a_1"), b2 = tmp("a_1");
	return L(local(id("_a"), num("0")), local(a1, num("1")), local(b1, num("2")), local(a2, num("3")), local(b2, num("4")), f(a1, a2, b1, b2));
});
ast("temp-shared-across-scopes", () => {
	// the same temp id seen in two scopes: the first visit decides the name
	const t = tmp("s"), u = tmp("s");
	return L(
		doBlock(local(id("_s"), num("1")), f(t)),
		doBlock(f(t)),
		f(u),
	);
});
ast("temp-return-in-function", () => {
	const t = tmp("r");
	return L(fnDecl(id("g"), [], [local(t, num("1")), ret(t)]));
});
add("temp-render-node", "render", () => {
	const t = tmp("n");
	return bin(t, "+", t);
});
add("temp-render-statements", "renderStatements", () => {
	const t = tmp("n"), u = tmp("n");
	return L(local(t, num("1")), local(u, num("2")), f(t, u));
});
add("temp-unsolved-fallback", "renderUnsolved", () => {
	const a = tmp("a"), b = tmp("b"), c = tmp("");
	return call(id("f"), a, b, c, a, b);
});
add("temp-unsolved-fallback-list", "renderStatementsUnsolved", () => {
	const a = tmp("a"), b = tmp("b");
	return L(local(a, num("1")), local(b, a), f(a, b));
});
add("temp-unsolved-mapfield-order", "renderUnsolved", () =>
	luau.map([[tmp("k"), tmp("v")], [tmp("k2"), luau.array([tmp("v2")])], [str("name"), tmp("v3")]]),
);
add("temp-unsolved-mapfield-nested", "renderUnsolved", () =>
	luau.map([[tmp("k"), luau.map([[tmp("k2"), tmp("v2")], [num("1"), tmp("v3")]])], [tmp("k3"), tmp("v4")]]),
);
add("temp-unsolved-mixedtable", "renderUnsolved", () =>
	luau.mixedTable([tmp("a"), [tmp("k"), tmp("v")], tmp("b")]),
);

// indentation ----------------------------------------------------------------------------------------------------------------
ast("indentation", () =>
	L(
		fnDecl(id("a"), [], [
			ifS(id("c"), [
				forS([id("k"), id("v")], id("t"), whileS(id("w"), local(id("m"), luau.map([[str("a"), luau.mixedTable([id("b"), [str("c"), fn([], [doBlock(f(luau.set([str("s")])))])]])]])))),
			], L(repeatS(id("d"), comment("multi\nline"), f(interp(part("x\n"), id("y")))))),
		]),
	),
);

// shared and cloned nodes (clone-on-adopt, readonly lists, list quirks) ------------------------------------------------------------
ast("shared-node-in-two-places", () => {
	const shared = bin(id("a"), "+", id("b"));
	return L(set(x(), bin(shared, "*", shared)), set(x(), un("-", shared)), set(x(), bin(id("c"), "-", shared)), local(id("l"), ifx(shared, shared, shared)));
});
ast("shared-precedence-parent-changes", () => {
	const inner = bin(id("a"), "or", id("b"));
	const first = bin(inner, "or", id("c")); // inner left of or: no parens, adopts inner
	const second = bin(id("c"), "or", inner); // inner right of or: clone, needs parens
	return L(set(x(), first), set(x(), second));
});
ast("shared-statement-in-two-blocks", () => {
	const s = f(id("shared"));
	return L(doBlock(s), doBlock(s, s), ifS(id("c"), [s], L(s)));
});
ast("shared-string-key", () => {
	const s = str("multi\nline");
	return L(set(index(id("t"), s), s), local(id("m"), luau.map([[s, s]])), local(id("st"), luau.set([s, s])));
});
ast("shared-temp-node", () => {
	const t = tmp("t");
	return L(local(t, num("1")), f(t), doBlock(f(t), local(t, t)));
});
ast("shared-clone-sees-original-grandchildren", () => {
	// A shallow clone keeps pointing at the original's children, whose parent is the original.
	const inner = bin(id("a"), "+", id("b"));
	const outer = bin(inner, "*", id("c"));
	const x1 = set(x(), outer);
	const x2 = set(x(), outer);
	return L(x1, x2);
});
ast("shared-function-body", () => {
	const fe = fn([id("p")], [local(tmp("t"), id("p")), ret(id("p"))]);
	return L(local(id("a"), fe), local(id("b"), fe), f(fe));
});
ast("list-pushlist", () => {
	const a = L(f(id("a1")));
	const b = L(f(id("b1")), f(id("b2")));
	luau.list.pushList(a, b);
	luau.list.push(a, f(id("a2")));
	return L(doBlock(), c(K.DoStatement, { statements: a }));
});
ast("list-unshiftlist", () => {
	const a = L(f(id("a1")));
	const b = L(f(id("b1")), f(id("b2")));
	luau.list.unshiftList(a, b);
	luau.list.unshift(a, f(id("a0")));
	return L(c(K.DoStatement, { statements: a }));
});
ast("list-push-unshift-shift", () => {
	const a = L();
	luau.list.push(a, f(id("n1")));
	luau.list.unshift(a, f(id("n0")));
	luau.list.push(a, f(id("n2")));
	luau.list.shift(a);
	luau.list.unshift(a, f(id("x")));
	return L(c(K.DoStatement, { statements: a }), c(K.DoStatement, { statements: L() }));
});
ast("list-shift-to-empty", () => {
	const a = L(f(id("n1")));
	luau.list.shift(a);
	luau.list.shift(a);
	luau.list.push(a, f(id("n2")));
	return L(c(K.DoStatement, { statements: a }));
});
ast("list-pushlist-empty-cases", () => {
	const e1 = L(), e2 = L(), n = L(f(id("n")));
	luau.list.pushList(e1, n);
	luau.list.pushList(e2, L());
	const u1 = L(), u2 = L(f(id("u")));
	luau.list.unshiftList(u1, u2);
	return L(c(K.DoStatement, { statements: e1 }), c(K.DoStatement, { statements: e2 }), c(K.DoStatement, { statements: u1 }));
});
ast("list-readonly-other-renders-chain", () => {
	// pushList marks `b` read-only but its head still points into the merged chain,
	// and the renderer follows `next` past `b.tail`.
	const a = L(f(id("a1")));
	const b = L(f(id("b1")));
	luau.list.pushList(a, b);
	luau.list.push(a, f(id("a2")));
	return L(c(K.DoStatement, { statements: b }), c(K.DoStatement, { statements: a }));
});
ast("list-clone", () => {
	const a = L(f(id("a1")), f(id("a2")));
	const d = c(K.DoStatement, { statements: a });
	const copy = luau.list.clone(a);
	return L(d, c(K.DoStatement, { statements: copy }));
});
ast("list-clone-scope-edge", () => {
	// cloned statements keep their old parent, so they are not the head/tail of the new block
	const t1 = tmp("v"), t2 = tmp("v");
	const a = L(local(t1, num("1")), local(t2, num("2")));
	const d = c(K.DoStatement, { statements: a });
	const copy = luau.list.clone(a);
	return L(d, local(id("_v"), num("3")), c(K.DoStatement, { statements: copy }));
});

// render() directly on nodes ----------------------------------------------------------------------------------------------------
add("render-expression-no-parent", "render", () => bin(num("1"), "+", num("2")));
add("render-expression-in-parent", "render", () => {
	const inner = bin(num("1"), "or", num("2"));
	bin(num("0"), "and", inner);
	return inner;
});
add("render-parenthesized-chain", "render", () => paren(paren(bin(id("a"), "+", id("b")))));
add("render-string-in-set", "render", () => luau.set([str('a"b\'c')]));
add("render-statement-alone", "render", () => local(id("a"), call(id("f"))));
add("render-statement-alone-semicolon-no-stack", "render", () => stmt(call(id("f"))));
add("render-function", "render", () => fn([id("a"), tmp("a")], [local(tmp("b"), id("a")), ret(tmp("b"))], true));
add("render-comment", "render", () => comment("a\nb"));
add("render-map-field", "render", () => c(K.MapField, { index: str("a b"), value: num("1") }));
add("render-interp-part", "render", () => part("{x}\n"));
add("render-if-statement", "render", () => ifS(id("a"), [f()], ifS(id("b"), [f()], L(f()))));
add("render-list-statements", "renderStatements", () => L(local(id("a"), num("1")), stmt(call(paren(id("f"))))));
add("render-list-empty", "renderStatements", () => L());
ast("render-ast-empty", () => L());

module.exports = { cases, helpers: { L, id, tmp, num, str, nil, varargs, prop, call, mcall, paren, bin, un, index, ifx, fn, interp, part, local, assign, set, stmt, ret, brk, cont, comment, doBlock, whileS, repeatS, ifS, forS, nfor, fnDecl, methodDecl, f, x, BINARY_OPS, UNARY_OPS, ASSIGN_OPS } };

// A read-only list rendered on its own still walks `next` past its tail.
ast("list-readonly-other-top-level", () => {
	const a = L(f(id("a1")));
	const b = L(f(id("b1")));
	luau.list.pushList(a, b);
	luau.list.push(a, f(id("a2")));
	return b;
});

// A multi-line string with no parent gets no bracket spacing.
add("render-string-no-parent", "render", () => str("a\nb"));
add("render-string-no-parent-both-quotes", "render", () => str("a'\"b"));
