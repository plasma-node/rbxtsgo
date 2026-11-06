// Generates the differential goldens from real JS roblox-ts@3.0.0 on TS 5.5.3. Public code only.
// Run through scripts/gen-diff-goldens.sh. Usage: node gen-golden.cjs <node_modules with roblox-ts> [corpus name...]
//
// "perfile" corpora do what upstream's src/CLI/test.ts does: one program, then compileFiles on one file at a time,
// so an error in one file can't stop the others. "whole" corpora call compileFiles once with every file, as the
// build command does. Output for each file goes to golden/<corpus>/<path under outDir>, and each file's diagnostics
// go to golden/<corpus>/diagnostics.json in compile order. Paths are relative to the project folder.
//
// With the writeTransformedFiles option, each file's reprinted text goes to golden/<corpus>/transformed/.
//
// "plugin-exit" corpora are for transformer plugins that write to stdout and call process.exit. Each file gets one
// process, and what that process wrote is recorded as one diagnostic with source "plugin".
process.env.NO_COLOR = "1";
delete process.env.FORCE_COLOR;
const { execFileSync, spawnSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const modules = path.resolve(process.argv[2]);
const only = process.argv.slice(3);
const repoRoot = path.resolve(__dirname, "..", "..");

const rt = name => require(path.join(modules, "roblox-ts", "out", ...name.split("/")));
const ts = require(path.join(modules, "typescript"));
const { compileFiles } = rt("Project/functions/compileFiles.js");
const { copyFiles } = rt("Project/functions/copyFiles.js");
const { copyInclude } = rt("Project/functions/copyInclude.js");
const { createPathTranslator } = rt("Project/functions/createPathTranslator.js");
const { createProjectData } = rt("Project/functions/createProjectData.js");
const { createProjectProgram } = rt("Project/functions/createProjectProgram.js");
const { getChangedSourceFiles } = rt("Project/functions/getChangedSourceFiles.js");
const { getRootDirs } = rt("Shared/util/getRootDirs.js");
const { getDiagnosticId } = rt("Shared/diagnostics.js");
const { DEFAULT_PROJECT_OPTIONS } = rt("Shared/constants.js");

const toSlash = p => p.split(path.sep).join("/");

function describeDiagnostic(d, projectDir) {
	const top = typeof d.messageText === "string" ? d.messageText : d.messageText.messageText;
	const entry = {
		source: typeof d.code === "number" ? "ts" : "rbxts",
		code: typeof d.code === "number" ? d.code : -1,
		id: getDiagnosticId(d) ?? -1,
		category: ts.DiagnosticCategory[d.category].toLowerCase(),
		message: top.replaceAll(projectDir, "<ROOT>").replaceAll(toSlash(projectDir), "<ROOT>"),
		file: "",
		line: -1,
		column: -1,
	};
	if (d.file) {
		entry.file = toSlash(path.relative(projectDir, d.file.fileName));
		if (d.start !== undefined) {
			const { line, character } = d.file.getLineAndCharacterOfPosition(d.start);
			entry.line = line;
			entry.column = character;
		}
	}
	return entry;
}

function collectTransformed(outDir, goldenDir) {
	const walk = dir => {
		if (!fs.existsSync(dir)) return;
		for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
			const full = path.join(dir, entry.name);
			if (entry.isDirectory()) {
				walk(full);
			} else if (/\.transformed\.tsx?$/.test(entry.name)) {
				const dest = path.join(goldenDir, "transformed", path.relative(outDir, full));
				fs.mkdirSync(path.dirname(dest), { recursive: true });
				fs.copyFileSync(full, dest);
			}
		}
	};
	walk(outDir);
}

const ansi = /\x1b\[[0-9;]*m/g;

function run(corpus) {
	const tsconfig = path.resolve(repoRoot, corpus.tsconfig);
	if (!fs.existsSync(tsconfig)) throw new Error(`missing ${tsconfig}: run scripts/setup-diff-corpus.sh`);
	const projectDir = path.dirname(tsconfig);
	const goldenDir = path.join(__dirname, "golden", corpus.name);
	fs.rmSync(goldenDir, { recursive: true, force: true });
	fs.mkdirSync(goldenDir, { recursive: true });

	const data = createProjectData(tsconfig, Object.assign({}, DEFAULT_PROJECT_OPTIONS, { project: "" }, corpus.options));
	const program = createProjectProgram(data);
	const pathTranslator = createPathTranslator(program, data);
	const outDir = program.getCompilerOptions().outDir;
	fs.rmSync(outDir, { recursive: true, force: true });
	copyInclude(data);
	copyFiles(data, pathTranslator, new Set(getRootDirs(program.getCompilerOptions())));

	const sourceFiles = getChangedSourceFiles(program);
	const diagnostics = {};
	const started = Date.now();
	const record = (list, key) => {
		if (list.length > 0) diagnostics[key] = list.map(d => describeDiagnostic(d, projectDir));
	};
	const written = new Set();
	const saveOutputs = (emitted, keyFile) => {
		for (const outPath of emitted) {
			const rel = toSlash(path.relative(outDir, outPath));
			const dest = path.join(goldenDir, "files", rel);
			fs.mkdirSync(path.dirname(dest), { recursive: true });
			fs.copyFileSync(outPath, dest);
			written.add(rel);
		}
	};
	if (process.env.GEN_GOLDEN_FILE) {
		// One file of a plugin-exit corpus. If the plugin exits, nothing below this runs.
		const sourceFile = sourceFiles.find(f => toSlash(path.relative(projectDir, f.fileName)) === process.env.GEN_GOLDEN_FILE);
		compileFiles(program.getProgram(), data, pathTranslator, [sourceFile]);
		return;
	}
	if (corpus.mode === "plugin-exit") {
		for (const sourceFile of sourceFiles) {
			const key = toSlash(path.relative(projectDir, sourceFile.fileName));
			const child = spawnSync(process.execPath, [__filename, modules], {
				env: { ...process.env, GEN_GOLDEN_CHILD: corpus.name, GEN_GOLDEN_FILE: key },
				encoding: "utf8",
			});
			if (child.status !== 0) {
				diagnostics[key] = [{ source: "plugin", code: child.status, id: -1, category: "error", message: child.stdout.replace(ansi, "").trimEnd(), file: "", line: -1, column: -1 }];
			}
		}
	} else if (corpus.mode === "perfile") {
		for (const sourceFile of sourceFiles) {
			const result = compileFiles(program.getProgram(), data, pathTranslator, [sourceFile]);
			record(result.diagnostics, toSlash(path.relative(projectDir, sourceFile.fileName)));
			saveOutputs(result.emittedFiles ?? []);
		}
	} else {
		const result = compileFiles(program.getProgram(), data, pathTranslator, sourceFiles);
		record(result.diagnostics, "<whole>");
		saveOutputs(result.emittedFiles ?? []);
	}
	const elapsed = Date.now() - started;
	collectTransformed(outDir, goldenDir);
	fs.writeFileSync(path.join(goldenDir, "diagnostics.json"), JSON.stringify(diagnostics, null, "\t") + "\n");
	const manifest = {
		sources: sourceFiles.map(f => toSlash(path.relative(projectDir, f.fileName))),
		outputs: [...written].sort(),
	};
	fs.writeFileSync(path.join(goldenDir, "manifest.json"), JSON.stringify(manifest, null, "\t") + "\n");
	console.log(`${corpus.name}: ${sourceFiles.length} sources, ${written.size} outputs, ${Object.keys(diagnostics).length} with diagnostics, compile ${elapsed} ms`);
}

// Each corpus gets its own process, because some diagnostics accumulate text in module-level state.
const corpora = JSON.parse(fs.readFileSync(path.join(__dirname, "corpora.json"), "utf8"));
if (process.env.GEN_GOLDEN_CHILD) {
	run(corpora.find(c => c.name === process.env.GEN_GOLDEN_CHILD));
} else {
	for (const corpus of corpora) {
		// A corpus with a "golden" field compares with another corpus's goldens, because JS can't read its config.
		if (!corpus.golden && (only.length === 0 || only.includes(corpus.name))) {
			execFileSync(process.execPath, [__filename, modules], {
				env: { ...process.env, GEN_GOLDEN_CHILD: corpus.name, NODE_PATH: modules },
				stdio: "inherit",
			});
		}
	}
}
