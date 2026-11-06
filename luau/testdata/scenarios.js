"use strict";
// Structural scenarios: they exercise create() adoption/cloning and the
// luau.list helpers (and the shared globals/strings nodes), then dump the
// resulting object graph. luau/scenarios_test.go replays the same steps with
// the Go API and must produce an identical graph, so these also pin the quirks
// (clone-on-adopt, read-only lists, dangling links) that the transformer
// depends on. Keep the two files in sync, and keep the globals scenarios first.

const luau = require("@roblox-ts/luau-ast").default;
const K = luau.SyntaxKind;
const c = luau.create;
const L = luau.list.make;
const id = luau.id;

// Each scenario returns { rootName: nodeOrList }.
const scenarios = [];
const scenario = (name, build) => scenarios.push({ name, build });

// These two must stay first: nothing may have touched the globals before.
scenario("globals-initial", () => {
	const roots = {};
	const walk = (prefix, obj) => {
		for (const key of Object.keys(obj).sort()) {
			const v = obj[key];
			if (luau.isNode(v)) roots[prefix + key] = v;
			else walk(prefix + key + ".", v);
		}
	};
	walk("", luau.globals);
	for (const key of Object.keys(luau.strings).sort()) roots["strings." + key] = luau.strings[key];
	// one global order: sorted by full name
	return Object.fromEntries(Object.entries(roots).sort((a, b) => (a[0] < b[0] ? -1 : 1)));
});
scenario("globals-first-use", () => {
	const first = luau.call(luau.globals.getmetatable, [id("a")]);
	const second = luau.call(luau.globals.getmetatable, [id("b")]);
	const str1 = luau.array([luau.strings.__mode]);
	const str2 = luau.array([luau.strings.__mode]);
	return { first, second, str1, str2, getmetatable: luau.globals.getmetatable, __mode: luau.strings.__mode };
});

scenario("create-adopts-children", () => {
	const a = id("a");
	const b = id("b");
	const call = luau.call(a, [b, id("c")]);
	return { call, a, b };
});
scenario("create-clones-reused-child", () => {
	const a = id("a");
	const sum = luau.binary(a, "+", a);
	return { sum, a };
});
scenario("create-clones-child-with-parent", () => {
	const shared = luau.binary(id("a"), "+", id("b"));
	const p1 = luau.unary("-", shared);
	const p2 = luau.unary("not", shared);
	return { p1, p2, shared };
});
scenario("create-clone-keeps-grandchildren-parent", () => {
	const inner = luau.binary(id("a"), "+", id("b"));
	const outer = luau.binary(inner, "*", id("c"));
	const s1 = c(K.Assignment, { left: id("x"), operator: "=", right: outer });
	const s2 = c(K.Assignment, { left: id("y"), operator: "=", right: outer });
	return { s1, s2, outer, inner };
});
scenario("create-list-elements", () => {
	const shared = id("s");
	const owner = c(K.DoStatement, {
		statements: L(c(K.CallStatement, { expression: luau.call(shared, []) }), c(K.CallStatement, { expression: luau.call(shared, []) })),
	});
	return { owner, shared };
});
scenario("create-list-element-with-parent", () => {
	const stmt = c(K.CallStatement, { expression: luau.call(id("f"), []) });
	const first = c(K.DoStatement, { statements: L(stmt) });
	const second = c(K.DoStatement, { statements: L(stmt) });
	return { first, second, stmt };
});
scenario("create-node-or-list-fields", () => {
	const left = L(id("a"), id("b"));
	const right = L(id("c"), id("d"));
	const assign = c(K.Assignment, { left, operator: "=", right });
	const decl = c(K.VariableDeclaration, { left: id("e"), right: undefined });
	const decl2 = c(K.VariableDeclaration, { left: L(id("f")), right: luau.nil() });
	const ret = c(K.ReturnStatement, { expression: L(id("g")) });
	const ifs = c(K.IfStatement, { condition: id("h"), statements: L(), elseBody: c(K.IfStatement, { condition: id("i"), statements: L(), elseBody: L(c(K.BreakStatement, {})) }) });
	return { assign, decl, decl2, ret, ifs };
});
scenario("create-optional-step", () => {
	const withStep = c(K.NumericForStatement, { id: id("i"), start: luau.number(1), end: luau.number(2), step: luau.number(3), statements: L() });
	const withoutStep = c(K.NumericForStatement, { id: id("i"), start: luau.number(1), end: luau.number(2), step: undefined, statements: L() });
	return { withStep, withoutStep };
});
scenario("helpers-constructors", () => ({
	none: luau.none(),
	nil: luau.nil(),
	t: luau.bool(true),
	f: luau.bool(false),
	n0: luau.number(0),
	n1: luau.number(1.5),
	n2: luau.number(-2),
	s: luau.string("s"),
	i: luau.id("i"),
	cm: luau.comment("c"),
	arr: luau.array([luau.number(1), luau.number(2)]),
	emptyArr: luau.array(),
	set: luau.set([luau.string("a")]),
	map: luau.map([[luau.string("k"), luau.number(1)], [luau.number(2), luau.nil()]]),
	mixed: luau.mixedTable([luau.nil(), [luau.string("k"), luau.number(1)]]),
	bin: luau.binary(luau.id("a"), "..", luau.id("b")),
	un: luau.unary("#", luau.id("a")),
	prop: luau.property(luau.id("a"), "b"),
	call: luau.call(luau.id("f"), [luau.id("a")]),
	callNoArgs: luau.call(luau.id("f")),
}));
scenario("list-make-and-push", () => {
	const a = L(id("1"), id("2"));
	luau.list.push(a, id("3"));
	const e = L();
	luau.list.push(e, id("only"));
	return { a, e };
});
scenario("list-shift-unshift", () => {
	const a = L(id("1"), id("2"), id("3"));
	const shifted = luau.list.shift(a);
	luau.list.unshift(a, id("0"));
	const one = L(id("solo"));
	const soloShifted = luau.list.shift(one);
	const empty = L();
	const emptyShifted = luau.list.shift(empty);
	luau.list.unshift(empty, id("u"));
	return { a, shifted, one, soloShifted, empty, __values: { emptyShifted: emptyShifted === undefined ? null : "node" } };
});
scenario("list-shift-dangling-next", () => {
	const a = L(id("1"), id("2"));
	const shifted = luau.list.shift(a);
	// the removed list node still points at "2"; reach it through a list built from the old head
	return { a, shifted };
});
scenario("list-pushlist", () => {
	const a = L(id("a1"), id("a2"));
	const b = L(id("b1"), id("b2"));
	luau.list.pushList(a, b);
	const e1 = L();
	const e2 = L(id("e2"));
	luau.list.pushList(e1, e2);
	const e3 = L(id("e3"));
	const e4 = L();
	luau.list.pushList(e3, e4);
	return { a, b, e1, e2, e3, e4 };
});
scenario("list-pushlist-then-push", () => {
	const a = L(id("a1"));
	const b = L(id("b1"));
	luau.list.pushList(a, b);
	luau.list.push(a, id("a2"));
	return { a, b };
});
scenario("list-unshiftlist", () => {
	const a = L(id("a1"), id("a2"));
	const b = L(id("b1"), id("b2"));
	luau.list.unshiftList(a, b);
	const e1 = L();
	const e2 = L(id("e2"));
	luau.list.unshiftList(e1, e2);
	return { a, b, e1, e2 };
});
scenario("list-unshiftlist-then-unshift", () => {
	const a = L(id("a1"));
	const b = L(id("b1"));
	luau.list.unshiftList(a, b);
	luau.list.unshift(a, id("a0"));
	return { a, b };
});
scenario("list-clone", () => {
	const owner = c(K.DoStatement, { statements: L(c(K.CallStatement, { expression: luau.call(id("f"), []) }), c(K.BreakStatement, {})) });
	const copy = luau.list.clone(owner.statements);
	return { owner, copy };
});
scenario("list-queries", () => {
	const a = L(id("1"), id("2"), id("3"));
	const arr = luau.list.toArray(a);
	const mapped = luau.list.mapToArray(a, n => n.name);
	return {
		a,
		arr0: arr[0],
		arr2: arr[2],
		__values: {
			size: luau.list.size(a),
			empty: luau.list.isEmpty(a),
			nonEmpty: luau.list.isNonEmpty(a),
			emptyList: luau.list.isEmpty(L()),
			every: luau.list.every(a, n => n.name !== "x"),
			everyFalse: luau.list.every(a, n => n.name === "1"),
			some: luau.list.some(a, n => n.name === "3"),
			someFalse: luau.list.some(a, n => n.name === "x"),
			mapped,
			arrLength: arr.length,
			isList: luau.list.isList(a),
			isListNode: luau.list.isList(id("a")),
			isListUndefined: luau.list.isList(undefined),
		},
	};
});
scenario("list-foreach-sees-appends", () => {
	const a = L(id("1"));
	const seen = [];
	luau.list.forEach(a, n => {
		seen.push(n.name);
		if (n.name.length < 3) luau.list.push(a, id(n.name + "x"));
	});
	return { a, __values: { seen } };
});

// Assertion messages thrown by the list helpers and renderer.
const errors = [];
const errorCase = (name, fn) => errors.push({ name, fn });
errorCase("list-push-readonly", () => {
	const a = L(id("a"));
	const b = L();
	luau.list.pushList(a, b);
	luau.list.push(b, id("x"));
});
errorCase("list-pushlist-to-readonly", () => {
	const a = L();
	const b = L();
	luau.list.pushList(a, b);
	luau.list.pushList(L(), b);
});
errorCase("list-pushlist-from-readonly-target", () => {
	const a = L();
	const b = L();
	luau.list.pushList(a, b);
	luau.list.pushList(b, L());
});
errorCase("list-shift-readonly", () => {
	const a = L(id("a"));
	luau.list.pushList(L(), a);
	luau.list.shift(a);
});
errorCase("list-unshift-readonly", () => {
	const a = L(id("a"));
	luau.list.pushList(L(), a);
	luau.list.unshift(a, id("b"));
});
errorCase("list-unshiftlist-readonly", () => {
	const a = L(id("a"));
	luau.list.unshiftList(L(), a);
	luau.list.unshiftList(L(), a);
});
errorCase("list-push-non-node", () => luau.list.push(L(), undefined));
errorCase("list-make-non-node", () => L(id("a"), undefined));

module.exports = { scenarios, errors };
