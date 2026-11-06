"use strict";
// Generates the golden files in golden/ using the real npm @roblox-ts/luau-ast.
//
//   node generate.js           write golden/*.json
//   node generate.js --check   exit 1 if the files on disk differ from a fresh run
//
// Files are written with LF line endings and are marked -text in
// .gitattributes so a checkout cannot change a byte.

const fs = require("fs");
const path = require("path");
const pkg = require("@roblox-ts/luau-ast");
const luau = pkg.default;
const { serialize, KIND_NAMES } = require("./graph");
const { cases } = require("./cases");
const { randomCases } = require("./random");
const { scenarios, errors } = require("./scenarios");

const VERSION = require("@roblox-ts/luau-ast/package.json").version;
if (VERSION !== "2.0.0") throw new Error(`expected @roblox-ts/luau-ast 2.0.0, found ${VERSION}`);

const GOLDEN_DIR = path.join(__dirname, "golden");

function runRender(mode, root) {
	const state = new pkg.RenderState();
	let expected;
	switch (mode) {
		case "renderAST":
			expected = pkg.renderAST(root);
			return { expected };
		case "renderStatements":
			pkg.solveTempIds(state, root);
			expected = pkg.renderStatements(state, root);
			break;
		case "render":
			pkg.solveTempIds(state, root);
			expected = pkg.render(state, root);
			break;
		case "renderUnsolved":
			expected = pkg.render(state, root);
			break;
		case "renderStatementsUnsolved":
			expected = pkg.renderStatements(state, root);
			break;
		default:
			throw new Error("unknown mode " + mode);
	}
	return { expected, temps: tempsOf(state) };
}

function tempsOf(state) {
	const out = {};
	for (const [key, value] of [...state.seenTempNodes].sort((a, b) => a[0] - b[0])) out[key] = value;
	return out;
}

function runCase({ name, mode, build }) {
	const root = build();
	const graph = serialize({ ast: root });
	const out = { name, mode, graph };
	try {
		Object.assign(out, runRender(mode, root));
		// renderAST does not expose its state; recompute the solved names for the golden.
		if (mode === "renderAST") {
			const state = new pkg.RenderState();
			pkg.solveTempIds(state, root);
			out.temps = tempsOf(state);
		}
	} catch (e) {
		if (!String(e.message).startsWith("Assertion Failed!")) throw new Error(`case ${name}: ${e.stack}`);
		out.error = e.message;
	}
	return out;
}

function stringify(header, items) {
	return JSON.stringify(header).replace(/\}$/, ',"cases":[\n') + items.map(i => JSON.stringify(i)).join(",\n") + "\n]}\n";
}

function renderFile(items) {
	return stringify({ luauAstVersion: VERSION }, items);
}

// util.json ------------------------------------------------------------------------------------------------------

function mulberry32(a) {
	return function () {
		a |= 0;
		a = (a + 0x6d2b79f5) | 0;
		let t = Math.imul(a ^ (a >>> 15), 1 | a);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

function bitsOf(value) {
	const buf = new DataView(new ArrayBuffer(8));
	buf.setFloat64(0, value);
	return buf.getBigUint64(0).toString(16).padStart(16, "0");
}

function buildUtil() {
	const rng = mulberry32(12345);
	const pick = arr => arr[Math.floor(rng() * arr.length)];
	const randomStrings = (alphabet, count, maxLen) =>
		Array.from({ length: count }, () => Array.from({ length: Math.floor(rng() * maxLen) + 1 }, () => pick(alphabet)).join(""));

	const identAlphabet = ["a", "z", "A", "Z", "_", "0", "9", "-", " ", ".", "é", "$", "e", "n", "d", "i", "f"];
	const identifiers = [
		"", "a", "_", "A1", "a_b", "0a", "a-b", "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "if", "in",
		"local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while", "continue", "goto", "self", "type", "Ends", "a\n", "\na", "é",
		...["and", "end", "nil", "function", "local"].flatMap(k => [k + "_", "_" + k, k.toUpperCase()]),
		...randomStrings(identAlphabet, 1500, 6),
	];

	const numAlphabet = ["0", "1", "7", "9", "_", ".", "e", "E", "+", "-", "x", "X", "b", "B", "o", "O", "a", "f", "F", "g", " ", "\n", "I", "n"];
	const numberStrings = [
		"0", "1", ".5", "5.", "1e5", "1e", "0x", "0b", "0x_", "0b_", "0_x1", "0x1_", "1_", "_1", "1._", "._1", ".", "..", "1.2.3", "1e+5", "1e+",
		"1e+_5", "1e_+5", "0X1F", "0B11", "0b12", "0o7", "Infinity", "-1", "+1", "1 ", " 1", "1\n", "00", "0.0", "0e0", "0E-0",
		...randomStrings(numAlphabet, 3000, 7),
	];

	const metamethods = [
		"__index", "__newindex", "__call", "__concat", "__unm", "__add", "__sub", "__mul", "__div", "__mod", "__pow", "__tostring",
		"__metatable", "__eq", "__lt", "__le", "__mode", "__gc", "__len", "__idiv", "__iter", "__namecall", "index", "__Index", "", "new",
		"__INDEX", "__len ", "__close",
	];
	const classFields = ["__index", "new", "New", "__newindex", "", "constructor", "__index ", "NEW"];
	const reserved = [
		"_G", "TS", "assert", "bit32", "coroutine", "error", "exports", "getmetatable", "ipairs", "next", "pairs", "pcall", "require", "script",
		"select", "self", "setmetatable", "string", "super", "table", "utf8", "math", "tostring", "type", "typeof", "unpack", "game", "yield",
		"min", "toString", "hasOwnProperty", "constructor", "__proto__", "valueOf", "", "G", "ts", "Game", "os", "print", "workspace", "byte",
	];

	const floats = [
		0, -0, 1, -1, 0.1, 0.5, 1.5, 123.456, 1e21, 1e-7, 1e-6, 1e20, 123456789012345680000, 2 ** 53, 2 ** 53 + 2, 2 ** 64, 2 ** 70,
		5e-324, 1.7976931348623157e308, Infinity, -Infinity, NaN, 0.000001, 0.0000001, 1 / 3, 2 / 3, 100, 1e5, 12345678.9, 4.35, 0.3, 123e-20,
		999999999999999900000, 1e300, -1.5e-10, 3.14159, 255, 0xffffffff, 4294967296, 1e15, 1e16, 1e17, 1.5e25, 9007199254740993,
	];
	for (let i = 0; i < 1500; i++) {
		const kind = i % 4;
		if (kind === 0) {
			// random bit patterns across the exponent range
			const hi = Math.floor(rng() * 0x7fe00000) | 0;
			const lo = Math.floor(rng() * 0xffffffff);
			const dv = new DataView(new ArrayBuffer(8));
			dv.setUint32(0, hi);
			dv.setUint32(4, lo);
			floats.push(dv.getFloat64(0));
		} else if (kind === 1) {
			floats.push(Math.floor(rng() * 1e9) / Math.pow(10, Math.floor(rng() * 12)));
		} else if (kind === 2) {
			floats.push(Math.floor(rng() * 1e15) * Math.pow(10, Math.floor(rng() * 12)));
		} else {
			floats.push(-Math.floor(rng() * 1e6) / 1000);
		}
	}

	const kindNames = [];
	for (let k = -1; k < 45; k++) kindNames.push([k, luau.getKindName(k) ?? null]);

	const numberLiterals = floats.map(v => {
		if (Number.isNaN(v)) return null; // luau.number(NaN) never terminates
		const node = luau.number(v);
		return [bitsOf(v), node.kind === luau.SyntaxKind.UnaryExpression ? { neg: true, value: node.expression.value } : { neg: false, value: node.value }];
	}).filter(Boolean);

	// String(Number(s.replace(/_/g, ""))) and Number(s) === 1, for strings and for the doubles
	const fallbacks = numberStrings.map(s => [s, String(Number(s.replace(/_/g, ""))), Number(s) === 1]);
	const toStrings = floats.map(v => [bitsOf(v), String(v)]);

	return {
		isValidIdentifier: identifiers.map(s => [s, luau.isValidIdentifier(s)]),
		isValidNumberLiteral: numberStrings.map(s => [s, luau.isValidNumberLiteral(s)]),
		isMetamethod: metamethods.map(s => [s, luau.isMetamethod(s)]),
		isReservedClassField: classFields.map(s => [s, luau.isReservedClassField(s)]),
		isReservedIdentifier: reserved.map(s => [s, luau.isReservedIdentifier(s)]),
		getKindName: kindNames,
		numberLiterals,
		numberFallbacks: fallbacks,
		numberToString: toStrings,
		guards: buildGuards(),
	};
}

// Every kind guard exported by the package, evaluated for every kind.
function buildGuards() {
	const names = Object.keys(luau).filter(k => /^(is[A-Z]|has[A-Z])/.test(k) && typeof luau[k] === "function");
	const kindGuards = names.filter(n => !["isNode", "isValidIdentifier", "isValidNumberLiteral", "isMetamethod", "isReservedClassField", "isReservedIdentifier"].includes(n));
	const out = {};
	for (const name of kindGuards) out[name] = KIND_NAMES.map((_, kind) => luau[name]({ kind }));
	return { kinds: KIND_NAMES, guards: out, isNode: [undefined, null, 5, "x", {}, { kind: "x" }, { kind: -1 }, { kind: 0 }, { kind: 39 }, { kind: 40 }, []].map(v => luau.isNode(v)) };
}

// scenarios.json ---------------------------------------------------------------------------------------------------

function buildScenarios() {
	const out = [];
	for (const { name, build } of scenarios) {
		const roots = build();
		const values = roots.__values;
		delete roots.__values;
		out.push({ name, graph: serialize(roots), values: values ?? null });
	}
	for (const { name, fn } of errors) {
		let message = null;
		try {
			fn();
		} catch (e) {
			message = e.message;
		}
		if (message === null || !message.startsWith("Assertion Failed!")) throw new Error(`error scenario ${name} did not assert`);
		out.push({ name, error: message });
	}
	return out;
}

// main ------------------------------------------------------------------------------------------------------------------

function buildAll() {
	const files = {};
	files["handwritten.json"] = renderFile(cases.map(runCase));
	files["random.json"] = renderFile(randomCases(160).map(runCase));
	files["scenarios.json"] = stringify({ luauAstVersion: VERSION }, buildScenarios());
	files["util.json"] = JSON.stringify({ luauAstVersion: VERSION, ...buildUtil() }) + "\n";
	return files;
}

const files = buildAll();
if (process.argv.includes("--check")) {
	let bad = 0;
	for (const [name, text] of Object.entries(files)) {
		const file = path.join(GOLDEN_DIR, name);
		const existing = fs.existsSync(file) ? fs.readFileSync(file, "utf8") : null;
		if (existing !== text) {
			console.error(`golden/${name} is out of date`);
			bad++;
		}
	}
	process.exit(bad ? 1 : 0);
} else {
	fs.mkdirSync(GOLDEN_DIR, { recursive: true });
	for (const [name, text] of Object.entries(files)) {
		fs.writeFileSync(path.join(GOLDEN_DIR, name), text, "utf8");
		const n = text.split("\n").length - 3;
		console.log(`wrote golden/${name} (${text.length} bytes)`);
	}
}
