"use strict";
// Deterministic random program generator. Programs are built with the real
// @roblox-ts/luau-ast API, including the things a transformer does: nodes and
// temporary identifiers reused in several places (clone on adopt), mixed
// list shapes, nested scopes. The seed makes every run produce the same set.

const luau = require("@roblox-ts/luau-ast").default;
const { helpers: h } = require("./cases");
const K = luau.SyntaxKind;
const c = luau.create;

function mulberry32(a) {
	return function () {
		a |= 0;
		a = (a + 0x6d2b79f5) | 0;
		let t = Math.imul(a ^ (a >>> 15), 1 | a);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

const NAMES = ["a", "b", "c", "x", "_x", "_x_1", "_", "_1", "t", "self", "_a", "a_1", "_t", "_t_1", "_t_2"];
const TEMP_NAMES = ["", "", "x", "t", "a", "a_1", "_", "v"];
const NUMBERS = ["0", "1", "2", "10", "1.5", "0x10", "1e3", "1_000", "0b11", ".5", "0o7", "+3", "", "NaN", "1e21", "+1e-7"];
const STRINGS = ["", "a", "hello world", 'say "hi"', "it's", `both "'`, "multi\nline", "]]", "x]=]", "ends]", "\\n", "é", "end", "a.b"];
const PART_TEXTS = ["", "text ", "{", "}", "{{}}", "\\u{1F600}", "a\nb", "a\r\nb", "\\", "x{y}z"];
const COMMENT_TEXTS = ["", "note", "two\nlines", "x]]y\nz", "end]="];
const PROP_NAMES = ["a", "b", "end", "bad name", "x1", "_", "then", "é"];
const METHODS = ["m", "go", "_x", "A1"];

function generate(seed, size) {
	const rng = mulberry32(seed);
	const chance = p => rng() < p;
	const pick = arr => arr[Math.floor(rng() * arr.length)];
	const range = (lo, hi) => lo + Math.floor(rng() * (hi - lo + 1));

	const exprPool = []; // nodes available for reuse (-> clone on adopt)
	const tempPool = [];

	function remember(e) {
		if (exprPool.length < 20) exprPool.push(e);
		return e;
	}

	function temp() {
		if (tempPool.length > 0 && chance(0.5)) return pick(tempPool);
		const t = h.tmp(pick(TEMP_NAMES));
		tempPool.push(t);
		return t;
	}

	function simple() {
		switch (range(0, 9)) {
			case 0: case 1: return h.id(pick(NAMES));
			case 2: case 3: return temp();
			case 4: return h.num(pick(NUMBERS));
			case 5: return h.str(pick(STRINGS));
			case 6: return luau.nil();
			case 7: return luau.bool(chance(0.5));
			case 8: return h.varargs();
			default: return luau.number(pick([0, 1, 2.5, -3, 1e21]));
		}
	}

	function indexable(d) {
		if (d <= 0) return chance(0.6) ? h.id(pick(NAMES)) : temp();
		switch (range(0, 6)) {
			case 0: return h.id(pick(NAMES));
			case 1: return temp();
			case 2: return h.prop(indexable(d - 1), pick(PROP_NAMES));
			case 3: return h.index(indexable(d - 1), expr(d - 1));
			case 4: return h.call(indexable(d - 1), ...exprs(d - 1, range(0, 2)));
			case 5: return h.mcall(indexable(d - 1), pick(METHODS), ...exprs(d - 1, range(0, 2)));
			default: return h.paren(expr(d - 1));
		}
	}

	function exprs(d, n) {
		return Array.from({ length: n }, () => expr(d));
	}

	function expr(d) {
		if (d <= 0) return simple();
		if (exprPool.length > 0 && chance(0.08)) return pick(exprPool);
		switch (range(0, 17)) {
			case 0: case 1: return remember(h.bin(expr(d - 1), pick(h.BINARY_OPS), expr(d - 1)));
			case 2: return remember(h.un(pick(h.UNARY_OPS), expr(d - 1)));
			case 3: return remember(h.ifx(expr(d - 1), expr(d - 1), chance(0.5) ? h.ifx(expr(d - 1), expr(d - 1), expr(d - 1)) : expr(d - 1)));
			case 4: case 5: return indexable(d);
			case 6: return h.paren(expr(d - 1));
			case 7: return luau.array(exprs(d - 1, range(0, 3)));
			case 8: return luau.map(Array.from({ length: range(0, 3) }, () => [chance(0.4) ? h.str(pick(PROP_NAMES)) : expr(d - 1), expr(d - 1)]));
			case 9: return luau.set(exprs(d - 1, range(0, 3)));
			case 10: return luau.mixedTable(Array.from({ length: range(0, 3) }, () => (chance(0.5) ? expr(d - 1) : [expr(d - 1), expr(d - 1)])));
			case 11: return func(d - 1);
			case 12: {
				const parts = Array.from({ length: range(0, 4) }, () => (chance(0.5) ? h.part(pick(PART_TEXTS)) : expr(d - 1)));
				return h.interp(...parts);
			}
			default: return simple();
		}
	}

	function anyId() {
		return chance(0.5) ? h.id(pick(NAMES)) : temp();
	}

	function writable(d) {
		switch (range(0, 3)) {
			case 0: return h.id(pick(NAMES));
			case 1: return temp();
			case 2: return h.prop(indexable(d), pick(PROP_NAMES));
			default: return h.index(indexable(d), expr(d));
		}
	}

	function params() {
		return Array.from({ length: range(0, 3) }, () => anyId());
	}

	function func(d) {
		return h.fn(params(), block(d, range(0, 3)), chance(0.3));
	}

	function listOrNode(make, d) {
		if (chance(0.3)) return luau.list.make(...Array.from({ length: range(1, 3) }, () => make(d)));
		return make(d);
	}

	function stmt(d) {
		switch (range(0, 17)) {
			case 0: case 1: return h.stmt(chance(0.6) ? h.call(indexable(d), ...exprs(d, range(0, 2))) : h.mcall(indexable(d), pick(METHODS), ...exprs(d, range(0, 2))));
			case 2: case 3: return h.local(chance(0.3) ? luau.list.make(anyId(), anyId()) : anyId(), chance(0.15) ? undefined : listOrNode(expr, d));
			case 4: case 5: return h.assign(chance(0.2) ? luau.list.make(writable(d), writable(d)) : writable(d), pick(h.ASSIGN_OPS), listOrNode(expr, d));
			case 6: return d > 0 ? h.doBlock(...block(d - 1, range(0, 3))) : h.comment(pick(COMMENT_TEXTS));
			case 7: return d > 0 ? h.whileS(expr(d - 1), ...block(d - 1, range(0, 3))) : h.comment(pick(COMMENT_TEXTS));
			case 8: return d > 0 ? h.repeatS(expr(d - 1), ...block(d - 1, range(0, 3))) : h.comment(pick(COMMENT_TEXTS));
			case 9: case 10: return d > 0 ? ifStmt(d - 1) : h.comment(pick(COMMENT_TEXTS));
			case 11: return d > 0 ? h.forS(Array.from({ length: range(0, 3) }, () => anyId()), expr(d - 1), ...block(d - 1, range(0, 3))) : h.comment("loop");
			case 12: return d > 0 ? h.nfor(anyId(), expr(d - 1), expr(d - 1), chance(0.5) ? undefined : (chance(0.5) ? h.num(pick(["1", "1.0", "2", "01"])) : expr(d - 1)), ...block(d - 1, range(0, 3))) : h.comment("nfor");
			case 13: return d > 0 ? h.fnDecl(chance(0.7) ? anyId() : h.prop(indexable(0), pick(PROP_NAMES)), params(), block(d - 1, range(0, 3)), false, chance(0.3)) : h.comment("fn");
			case 14: return d > 0 ? h.methodDecl(indexable(0), pick(METHODS), params(), block(d - 1, range(0, 3)), chance(0.3)) : h.comment("method");
			default: return h.comment(pick(COMMENT_TEXTS));
		}
	}

	function ifStmt(d) {
		let elseBody;
		switch (range(0, 3)) {
			case 0: elseBody = luau.list.make(); break;
			case 1: elseBody = luau.list.make(...block(d, range(1, 2))); break;
			case 2: elseBody = ifStmt(d > 0 ? d - 1 : 0); break;
			default: elseBody = luau.list.make();
		}
		return c(K.IfStatement, { condition: expr(d), statements: luau.list.make(...block(d, range(0, 3))), elseBody });
	}

	function block(d, n) {
		const out = Array.from({ length: n }, () => stmt(d));
		if (chance(0.15)) {
			out.push(pick([() => h.ret(chance(0.3) ? luau.list.make(expr(d), expr(d)) : expr(d)), () => h.brk(), () => h.cont()])());
			if (chance(0.3)) out.push(h.comment(pick(COMMENT_TEXTS)));
		}
		return out;
	}

	const top = Array.from({ length: size }, () => stmt(3));
	if (chance(0.2)) top.push(h.ret(expr(2)));
	return luau.list.make(...top);
}

function randomCases(count) {
	const out = [];
	for (let i = 0; i < count; i++) {
		const seed = 0x5eed0000 + i;
		out.push({ name: `random-${i}`, mode: i % 5 === 4 ? "renderStatements" : "renderAST", build: () => generate(seed, 4 + (i % 6)) });
	}
	return out;
}

module.exports = { randomCases };
