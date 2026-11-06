// Generates the TS 5.5.3 side of the checker and AST probe, on the public fixture in spike/fixture.
// Run through scripts/gen-goldens.sh. The Go side is probe_test.go, which must walk and print the same way.
// Usage: node gen-probe.cjs <path to typescript package>
const fs = require("fs");
const path = require("path");
const ts = require(path.resolve(process.argv[2]));

for (const [fixture, outName] of [
	[path.resolve(__dirname, "..", "..", "spike", "fixture"), "probe.txt"],
	[path.join(__dirname, "syntax"), "probe-syntax.txt"],
]) {
const parsed = ts.getParsedCommandLineOfConfigFile(path.join(fixture, "tsconfig.json"), {}, {
	...ts.sys,
	onUnRecoverableConfigFileDiagnostic: d => {
		throw new Error(String(d.messageText));
	},
});
const program = ts.createProgram({ rootNames: parsed.fileNames, options: parsed.options });
const c = program.getTypeChecker();

const TYPE_FLAGS = ["Any", "Unknown", "String", "Number", "Boolean", "Enum", "BigInt", "StringLiteral", "NumberLiteral", "BooleanLiteral", "EnumLiteral", "Undefined", "Null", "Void", "Never", "TypeParameter", "Object", "Union", "Intersection", "Index", "IndexedAccess", "Conditional", "TemplateLiteral"];
const SYMBOL_FLAGS = ["Variable", "Property", "Function", "Class", "Interface", "Enum", "EnumMember", "ValueModule", "NamespaceModule", "TypeLiteral", "ObjectLiteral", "Method", "Accessor", "Signature", "TypeParameter", "TypeAlias", "Alias", "Optional"];
const typeFlags = t => TYPE_FLAGS.filter(n => t.flags & ts.TypeFlags[n]).join("|");
const symbolFlags = s => SYMBOL_FLAGS.filter(n => s.flags & ts.SymbolFlags[n]).join("|");
const q = v => JSON.stringify(v);

function describeType(t) {
	if (!t) return "none";
	const symbol = t.symbol;
	return `${q(c.typeToString(t))} flags=${typeFlags(t)} symbol=${symbol ? symbol.name : "-"}`;
}

function describeSymbol(s) {
	if (!s) return "none";
	const decls = s.declarations ? s.declarations.length : 0;
	return `${s.name} flags=${symbolFlags(s)} decls=${decls} value=${!!s.valueDeclaration} parent=${s.parent ? s.parent.name : "-"}`;
}

const lines = [];
function visit(file, node) {
	const start = node.getStart(file);
	const here = `${start}-${node.end}`;
	if (ts.isIdentifier(node)) {
		lines.push(`${here} id ${q(node.text)}`);
		const s = c.getSymbolAtLocation(node);
		lines.push(`  symbol ${describeSymbol(s)}`);
		if (s) {
			lines.push(`  undefined=${c.isUndefinedSymbol(s)} arguments=${c.isArgumentsSymbol(s)} merged=${c.getMergedSymbol(s) === s}`);
			if (s.flags & ts.SymbolFlags.Alias) lines.push(`  skipAlias ${describeSymbol(ts.skipAlias(s, c))}`);
		}
	}
	if (ts.isStringLiteral(node) || ts.isNumericLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
		lines.push(`${here} literal ${q(node.text)}`);
	}
	if (ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node)) {
		lines.push(`${here} template ${q(node.text)} raw ${q(node.rawText)}`);
	}
	if (ts.isJsxText(node)) {
		lines.push(`${here} jsxtext ${q(node.text)} ws=${node.containsOnlyTriviaWhiteSpaces}`);
	}
	if (ts.isStatement(node)) {
		const comments = ts.getLeadingCommentRanges(file.text, node.pos) ?? [];
		if (comments.length) lines.push(`${here} comments ${q(comments.map(r => file.text.slice(r.pos, r.end)))}`);
	}
	if (ts.isExpressionNode(node) && !(ts.isFunctionExpression(node.parent) && node.parent.name === node)) {
		const t = c.getTypeAtLocation(node);
		lines.push(`${here} expr ${q(node.getText(file).slice(0, 40))} ${describeType(t)}`);
		lines.push(`  tuple=${c.isTupleType(t)} arrayLike=${c.isArrayLikeType(t)} calls=${t.getCallSignatures().length} constructs=${t.getConstructSignatures().length} props=${t.getProperties().length}`);
		lines.push(`  nonOptional ${q(c.typeToString(c.getNonOptionalType(t)))}`);
		const index = c.getIndexTypeOfType(t, ts.IndexKind.Number);
		lines.push(`  numberIndex ${index ? q(c.typeToString(index)) : "none"}`);
		const contextual = c.getContextualType(node);
		lines.push(`  contextual ${contextual ? q(c.typeToString(contextual)) : "none"}`);
		if (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node)) {
			lines.push(`  constant ${q(c.getConstantValue(node) ?? null)}`);
		}
		if (ts.isCallExpression(node) || ts.isNewExpression(node)) {
			const signature = c.getResolvedSignature(node);
			lines.push(`  signature params=${signature ? signature.parameters.length : -1} returns ${signature ? q(c.typeToString(signature.getReturnType())) : "none"}`);
		}
	}
	ts.forEachChild(node, child => visit(file, child));
}

const toSlash = p => p.split(path.sep).join("/");
for (const file of program.getSourceFiles()) {
	if (file.isDeclarationFile) continue;
	lines.push(`== ${toSlash(path.relative(fixture, file.fileName))}`);
	visit(file, file);
}
// The fixture folder is written as <ROOT> so the golden holds no machine path. probe_test.go does the same.
const root = toSlash(fixture);
fs.writeFileSync(path.join(__dirname, outName), lines.map(line => line.split(root).join("<ROOT>")).join("\n") + "\n");
}
