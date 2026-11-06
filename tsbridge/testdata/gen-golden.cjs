// Generates the TS 5.5.3 side of the bridge's differential tests. Run through scripts/gen-goldens.sh.
// Usage: node gen-golden.cjs <path to typescript package>
const fs = require("fs");

const path = require("path");
const ts = require(path.resolve(process.argv[2]));
if (ts.version !== "5.5.3") throw new Error("expected TypeScript 5.5.3, got " + ts.version);

const here = __dirname;
const scratch = path.resolve(here, "..", "..", ".scratch");
fs.mkdirSync(scratch, { recursive: true });
const temp = fs.mkdtempSync(path.join(scratch, "bridge-golden-"));
fs.cpSync(path.join(here, "project"), temp, { recursive: true });
fs.renameSync(path.join(temp, "_node_modules"), path.join(temp, "node_modules"));
const toSlash = p => p.split(path.sep).join("/");
const rel = p => toSlash(path.relative(temp, p));

function createProgram(configName) {
	const parsed = ts.getParsedCommandLineOfConfigFile(path.join(temp, configName), {}, {
		...ts.sys,
		onUnRecoverableConfigFileDiagnostic: d => {
			throw new Error(String(d.messageText));
		},
	});
	return ts.createProgram({ rootNames: parsed.fileNames, options: parsed.options });
}

// Same selection as getChangedSourceFiles, in program order.
const toCompile = program =>
	program.getSourceFiles().filter(f => !f.isDeclarationFile && !ts.isJsonSourceFile(f));

function format(d) {
	const text = ts.flattenDiagnosticMessageText(d.messageText, "\n");
	if (!d.file) return `TS${d.code}: ${text}`;
	const { line, character } = d.file.getLineAndCharacterOfPosition(d.start);
	return `${rel(d.file.fileName)}(${line + 1},${character + 1}): TS${d.code}: ${text}`;
}

function aliases(program) {
	const checker = program.getTypeChecker();
	const lines = [];
	for (const file of toCompile(program)) {
		const resolver = checker.getEmitResolver(file);
		const report = (node, label) =>
			lines.push(`${rel(file.fileName)}\t${label}\t${resolver.isReferencedAliasDeclaration(node)}`);
		for (const statement of file.statements) {
			if (!ts.isImportDeclaration(statement) || !statement.importClause) continue;
			const clause = statement.importClause;
			if (clause.name) report(clause, clause.name.text);
			const bindings = clause.namedBindings;
			if (bindings && ts.isNamedImports(bindings)) {
				for (const element of bindings.elements) report(element, element.name.text);
			} else if (bindings) {
				report(bindings, "* as " + bindings.name.text);
			}
		}
	}
	return lines;
}

function diagnostics(program) {
	const lines = [];
	for (const file of toCompile(program)) {
		for (const d of ts.getPreEmitDiagnostics(program, file)) lines.push(format(d));
	}
	return lines;
}

const out = path.join(here, "golden");
fs.mkdirSync(out, { recursive: true });
const write = (name, lines) => fs.writeFileSync(path.join(out, name), lines.join("\n") + "\n");
const main = createProgram("tsconfig.json");
write("aliases.txt", aliases(main));
write("diagnostics.txt", diagnostics(main));
write("files.txt", main.getSourceFiles().map(f => rel(f.fileName)));
const errors = createProgram("tsconfig.errors.json");
write("errors.txt", diagnostics(errors));
const rootDirs = createProgram("rootdirs/tsconfig.json");
write("rootdirs.txt", diagnostics(rootDirs));
write("rootdirs-common.txt", [rel(rootDirs.getCommonSourceDirectory())]);
fs.rmSync(temp, { recursive: true, force: true });
