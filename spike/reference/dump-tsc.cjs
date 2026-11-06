// Dumps the file list, module resolutions and diagnostics from a JS TypeScript,
// in the same format as tsgo-check's -dump-resolutions and -dump-diagnostics.
// Usage: node dump-tsc.cjs <path to typescript package> <tsconfig> <resolutions out> <diagnostics out>
const path = require("path");
const fs = require("fs");
const [tsPath, configArg, resOut, diagOut] = process.argv.slice(2);
const ts = require(path.resolve(tsPath));
const configPath = path.resolve(configArg);
const projectDir = path.dirname(configPath);
const toSlash = p => p.split(path.sep).join("/");
const parsed = ts.getParsedCommandLineOfConfigFile(configPath, {}, { ...ts.sys, onUnRecoverableConfigFileDiagnostic: d => { throw new Error(ts.flattenDiagnosticMessageText(d.messageText, "\n")); } });
const program = ts.createProgram({ rootNames: parsed.fileNames, options: parsed.options });
const rel = p => toSlash(path.relative(projectDir, p));
const lines = [];
for (const file of program.getSourceFiles()) {
	const name = rel(file.fileName);
	lines.push("file\t" + name);
	for (const spec of file.imports ?? []) {
		const mode = program.getModeForUsageLocation ? program.getModeForUsageLocation(file, spec) : undefined;
		const r = program.getResolvedModule
			? program.getResolvedModule(file, spec.text, mode)?.resolvedModule
			: ts.resolveModuleName(spec.text, file.fileName, parsed.options, ts.sys).resolvedModule;
		lines.push(`import\t${name}\t${spec.text}\t${r?.resolvedFileName ? rel(r.resolvedFileName) : "UNRESOLVED"}`);
	}
}
const sorted = [...new Set(lines)].sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
fs.writeFileSync(resOut, sorted.join("\n") + "\n");
const host = { getCurrentDirectory: () => projectDir, getCanonicalFileName: f => f, getNewLine: () => "\n" };
const diags = ts.sortAndDeduplicateDiagnostics([
	...program.getConfigFileParsingDiagnostics(),
	...program.getOptionsDiagnostics(),
	...program.getGlobalDiagnostics(),
	...program.getSyntacticDiagnostics(),
	...program.getSemanticDiagnostics(),
]);
fs.writeFileSync(diagOut, ts.formatDiagnostics(diags, host));
