// Generates golden output from the real roblox-ts@3.0.0 package's Shared code. Run through scripts/gen-goldens.sh.
// Usage: node gen-golden.cjs <node_modules with roblox-ts>
//
// kleur reads the environment at load, so each color mode runs in its own child process.
const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const modules = path.resolve(process.argv[2]);

if (process.argv[3] !== "child") {
	const out = {};
	for (const [name, env] of [
		["color", { FORCE_COLOR: "1" }],
		["plain", { NO_COLOR: "1" }],
	]) {
		const childEnv = { ...process.env, ...env };
		if (name === "color") delete childEnv.NO_COLOR;
		else delete childEnv.FORCE_COLOR;
		out[name] = JSON.parse(execFileSync(process.execPath, [__filename, modules, "child"], { env: childEnv, cwd: __dirname, encoding: "utf8", maxBuffer: 1 << 26 }));
	}
	const flavor = process.platform === "win32" ? "windows" : process.platform;
	fs.writeFileSync(path.join(__dirname, `golden.${flavor}.json`), JSON.stringify(out, null, "\t") + "\n");
	process.exit(0);
}

const shared = path.join(modules, "roblox-ts", "out", "Shared");
const ts = require(path.join(modules, "typescript"));
const { errors, warnings, getDiagnosticId } = require(path.join(shared, "diagnostics.js"));
const { formatDiagnostics } = require(path.join(shared, "util", "formatDiagnostics.js"));
const { createTextDiagnostic } = require(path.join(shared, "util", "createTextDiagnostic.js"));

const sourceText = "// leading comment\nconst value = 1;\n  // @ts-ignore\nconst other = 'x';\nexport { value, other };\n";
const file = ts.createSourceFile("sample.ts", sourceText, ts.ScriptTarget.ESNext, true);
const node = file.statements[1];
const located = { sourceFile: file, range: { pos: 19, end: 24 } };
const atEnd = { sourceFile: file, range: { pos: sourceText.length - 3, end: sourceText.length } };

const out = { factories: {}, context: {}, text: {}, format: {}, kinds: {} };
const describe = d => ({ message: d.messageText, id: getDiagnosticId(d) ?? null, category: d.category, start: d.start ?? null, length: d.length ?? null, code: d.code });
for (const [key, factory] of Object.entries(errors)) {
	if (typeof factory === "function" && factory.id !== undefined && key !== "noRojoData" && key !== "noPackageImportWithoutScope") {
		out.factories[key] = { factoryId: factory.id, atRange: describe(factory(located)), atNode: describe(factory(node)) };
	}
}
out.factories.runtimeLibUsedInReplicatedFirst = {
	factoryId: warnings.runtimeLibUsedInReplicatedFirst.id,
	atRange: describe(warnings.runtimeLibUsedInReplicatedFirst(located)),
	atNode: describe(warnings.runtimeLibUsedInReplicatedFirst(node)),
};
out.context.noRojoDataId = errors.noRojoData.id;
out.context.noPackageImportWithoutScopeId = errors.noPackageImportWithoutScope.id;
// Called in sequence on purpose: upstream's factories keep what earlier calls added.
out.context.noRojoData = [
	describe(errors.noRojoData(located, "first/path.lua", false)),
	describe(errors.noRojoData(located, "second/path.lua", true)),
	describe(errors.noRojoData(node, "third/path.lua", false)),
];
out.context.noPackageImportWithoutScope = [
	describe(errors.noPackageImportWithoutScope(located, "pkg/a.lua", ["ReplicatedStorage", "a"])),
	describe(errors.noPackageImportWithoutScope(node, "pkg/b.lua", ["ReplicatedStorage", "b", "c"])),
];
out.text.incorrectFileName = describe(errors.incorrectFileName("init.ts", "index.ts", "/full/path/init.ts"));
out.text.rojoPathInSrc = describe(errors.rojoPathInSrc("src/x", "out/x"));
out.text.transformerNotFound = describe(warnings.transformerNotFound("my-transformer", "Cannot find module"));
const truthy = warnings.truthyChange("0, \"\"");
const offset = warnings.stringOffsetChange("text");
out.text.truthyChange = { factoryId: truthy.id, diagnostic: describe(truthy(located)) };
out.text.stringOffsetChange = { factoryId: offset.id, diagnostic: describe(offset(located)) };

out.format.located = formatDiagnostics([errors.noNullLiteral(node)]);
out.format.range = formatDiagnostics([errors.noCommentDirectives(located)]);
out.format.rangeAtEnd = formatDiagnostics([errors.noCommentDirectives(atEnd)]);
out.format.warning = formatDiagnostics([warnings.runtimeLibUsedInReplicatedFirst(node)]);
out.format.text = formatDiagnostics([createTextDiagnostic("Just text")]);
out.format.several = formatDiagnostics([errors.noVar(node), createTextDiagnostic("Text after"), errors.noAny(file.statements[2])]);
out.sourceText = sourceText;
process.stdout.write(JSON.stringify(out));
