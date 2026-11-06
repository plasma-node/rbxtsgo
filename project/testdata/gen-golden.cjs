// Generates golden output from the real roblox-ts@3.0.0 package's Project code. Run through scripts/gen-goldens.sh.
// Usage: node gen-golden.cjs <node_modules with roblox-ts>
//
// kleur reads the environment at load, so each color mode runs in its own child process. Scratch folders go in
// .scratch, and every path in the output is relative to a <ROOT> or <TEMP> marker.
const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const modules = path.resolve(process.argv[2]);

const repoRoot = path.resolve(__dirname, "..", "..");
const scratch = path.join(repoRoot, ".scratch", "project-golden");

if (process.argv[3] !== "child") {
	const out = {};
	for (const [name, env] of [
		["color", { FORCE_COLOR: "1" }],
		["plain", { NO_COLOR: "1" }],
	]) {
		const childEnv = { ...process.env, ...env };
		if (name === "color") delete childEnv.NO_COLOR;
		else delete childEnv.FORCE_COLOR;
		const text = execFileSync(process.execPath, [__filename, modules, "child"], { env: childEnv, cwd: __dirname, encoding: "utf8", maxBuffer: 1 << 26 });
		out[name] = JSON.parse(text);
	}
	fs.rmSync(scratch, { recursive: true, force: true });
	const flavor = process.platform === "win32" ? "windows" : process.platform;
	fs.writeFileSync(path.join(__dirname, `golden.${flavor}.json`), JSON.stringify(out, null, "\t") + "\n");
	process.exit(0);
}

const rt = name => require(path.join(modules, "roblox-ts", "out", ...name.split("/")));
const ts = require(path.join(modules, "typescript"));
const { getParsedCommandLine } = rt("Project/functions/getParsedCommandLine.js");
const { createProjectData } = rt("Project/functions/createProjectData.js");
const { createNodeModulesPathMapping } = rt("Project/functions/createNodeModulesPathMapping.js");
const { copyItem } = rt("Project/functions/copyItem.js");
const { copyInclude } = rt("Project/functions/copyInclude.js");
const { cleanup } = rt("Project/functions/cleanup.js");
const { checkFileName } = rt("Project/functions/checkFileName.js");
const { checkRojoConfig } = rt("Project/functions/checkRojoConfig.js");
const { fileUsesCommentDirectives } = rt("Project/preEmitDiagnostics/fileUsesCommentDirectives.js");
const { createProjectProgram } = rt("Project/functions/createProjectProgram.js");
const { createPathTranslator } = rt("Project/functions/createPathTranslator.js");
const { compileFiles } = rt("Project/functions/compileFiles.js");
const { DiagnosticService } = rt("TSTransformer/classes/DiagnosticService.js");
const { LogService } = rt("Shared/classes/LogService.js");
const { DEFAULT_PROJECT_OPTIONS } = rt("Shared/constants.js");
const { findAncestorDir } = rt("Shared/util/findAncestorDir.js");
const { isPathDescendantOf } = rt("Shared/util/isPathDescendantOf.js");
const { RojoResolver } = require(path.join(modules, "@roblox-ts", "rojo-resolver"));
const { PathTranslator } = require(path.join(modules, "@roblox-ts", "path-translator"));

const root = path.join(__dirname, "fixtures");
const logs = [];
LogService.writeLine = (...messages) => logs.push(...messages.map(String));
LogService.write = message => logs.push(String(message));

const slashRoot = root.split(path.sep).join("/");
const clean = text => String(text).split(root).join("<ROOT>").split(slashRoot).join("<ROOT>").split(scratch).join("<TEMP>");
const fixture = (...parts) => path.join(root, ...parts);
const options = extra => ({ ...DEFAULT_PROJECT_OPTIONS, ...extra });

function tree(dir) {
	const lines = [];
	const walk = (current, prefix) => {
		if (!fs.existsSync(current)) return;
		for (const name of fs.readdirSync(current).sort()) {
			const full = path.join(current, name);
			if (fs.statSync(full).isDirectory()) {
				lines.push(`${prefix}${name}/`);
				walk(full, `${prefix}${name}/`);
			} else {
				lines.push(`${prefix}${name} = ${fs.readFileSync(full).toString("base64")}`);
			}
		}
	};
	walk(dir, "");
	return lines;
}

function freshDir(name) {
	const dir = path.join(scratch, name);
	fs.rmSync(dir, { recursive: true, force: true });
	fs.mkdirSync(dir, { recursive: true });
	return dir;
}

const write = (file, text) => {
	fs.mkdirSync(path.dirname(file), { recursive: true });
	fs.writeFileSync(file, text);
};

const out = {};

// tsconfig validation
out.validate = {};
for (const name of fs.readdirSync(fixture("validate")).filter(v => v.endsWith(".json"))) {
	try {
		getParsedCommandLine({ tsConfigPath: fixture("validate", name), projectPath: fixture("validate") });
		out.validate[name] = "ok";
	} catch (e) {
		out.validate[name] = clean(e.diagnostics.map(d => d.messageText).join("\n"));
	}
}

// createProjectData
out.projectData = {};
const dataCases = {
	app: ["data/app/tsconfig.json", {}],
	pkg: ["data/pkg/tsconfig.json", {}],
	nested: ["data/nested/deep/er/tsconfig.json", {}],
	rojos: ["data/rojos/tsconfig.json", {}],
	noname: ["data/noname/tsconfig.json", {}],
	badjson: ["data/badjson/tsconfig.json", {}],
	rojoOption: ["data/app/tsconfig.json", { rojo: fixture("data", "rojos", "a.project.json") }],
	includeOption: ["data/app/tsconfig.json", { includePath: fixture("data", "custom", "..", "include2") }],
	typeOption: ["data/pkg/tsconfig.json", { type: "game", noInclude: true }],
};
for (const [name, [rel, extra]] of Object.entries(dataCases)) {
	logs.length = 0;
	const data = createProjectData(fixture(...rel.split("/")), options(extra));
	out.projectData[name] = {
		isPackage: data.isPackage,
		nodeModulesPath: clean(data.nodeModulesPath),
		projectPath: clean(data.projectPath),
		rojoConfigPath: data.rojoConfigPath === undefined ? null : clean(data.rojoConfigPath),
		includePath: clean(data.projectOptions.includePath),
		logs: logs.map(clean),
	};
}

// createNodeModulesPathMapping
const mapping = createNodeModulesPathMapping([fixture("mapping", "@rbxts"), fixture("mapping", "missing")]);
const lowerRoot = root.toLowerCase();
const cleanKey = key => key.split(lowerRoot).join("<ROOT>").split(root).join("<ROOT>");
out.mapping = [...mapping.entries()].map(([k, v]) => `${cleanKey(k)} -> ${clean(v)}`).sort();

// copyItem
out.copy = {};
for (const declaration of [false, true]) {
	for (const writeOnlyChanged of [false, true]) {
		for (const existing of [false, true]) {
			const name = `decl=${declaration},changed=${writeOnlyChanged},existing=${existing}`;
			const dir = freshDir("copy");
			const outDir = path.join(dir, "out");
			if (existing) {
				write(path.join(outDir, "f.txt"), "old content\n");
				write(path.join(outDir, "d.lua"), "content of copy/src/d.lua\n");
				write(path.join(outDir, "sub", "h.luau"), "old\n");
				write(path.join(outDir, "stale.txt"), "stays\n");
				write(path.join(outDir, "sub", "stale", "x.lua"), "stays\n");
			}
			const translator = new PathTranslator(fixture("copy", "src"), outDir, undefined, declaration, true);
			const data = { projectOptions: options({ writeOnlyChanged }) };
			copyItem(data, translator, fixture("copy", "src"));
			out.copy[name] = tree(outDir);
		}
	}
}

// copyInclude
out.include = {};
for (const [name, extra, isPackage] of [
	["default", {}, false],
	["noInclude", { noInclude: true }, false],
	["typePackage", { type: "package" }, false],
	["isPackage", {}, true],
	["isPackageTypeModel", { type: "model" }, true],
]) {
	const dir = freshDir("include");
	const includePath = path.join(dir, "inc", "lower");
	copyInclude({ isPackage, projectOptions: options({ ...extra, includePath }) });
	out.include[name] = tree(dir);
}
{
	const dir = freshDir("include");
	const includePath = path.join(dir, "inc");
	write(path.join(includePath, "RuntimeLib.lua"), "overwritten\n");
	write(path.join(includePath, "Extra.lua"), "stays\n");
	copyInclude({ isPackage: false, projectOptions: options({ includePath }) });
	out.include.overwrite = tree(dir);
}

// cleanup
out.cleanup = {};
for (const declaration of [false, true]) {
	for (const luau of [false, true]) {
		const dir = freshDir("cleanup");
		const src = path.join(dir, "src");
		const outDir = path.join(dir, "out");
		for (const file of ["a.ts", "b.tsx", "index.ts", "sub/c.ts", "keep.txt", "sub/index.tsx", "typed.d.ts"]) write(path.join(src, file), "x\n");
		for (const file of [
			"a.lua", "a.luau", "b.lua", "b.luau", "init.lua", "init.luau", "index.lua", "orphan.lua", "orphan.luau", "keep.txt", "gone.txt",
			"sub/c.lua", "sub/c.luau", "sub/init.lua", "sub/init.luau", "sub/orphan.lua", "empty/gone.lua", "a.d.ts", "orphan.d.ts", "typed.d.ts", "tsconfig.tsbuildinfo", ".git/keep.lua",
		]) {
			write(path.join(outDir, file), "x\n");
		}
		const translator = new PathTranslator(src, outDir, path.join(outDir, "tsconfig.tsbuildinfo"), declaration, luau);
		cleanup(translator);
		out.cleanup[`decl=${declaration},luau=${luau}`] = tree(outDir).map(v => v.split(" = ")[0]);
	}
}

// checkFileName
out.fileNames = {};
for (const name of ["init.ts", "init.tsx", "init.d.ts", "init.server.ts", "init.server.tsx", "init.server.d.ts", "init.client.ts", "init.client.tsx", "init.client.d.ts", "index.ts", "init.lua", "init.foo.ts", "Init.ts"]) {
	const filePath = "/some/dir/" + name;
	checkFileName(filePath);
	out.fileNames[name] = DiagnosticService.flush().map(d => d.messageText);
}

// checkRojoConfig
{
	const configPath = fixture("rojocheck", "default.project.json");
	const data = { rojoConfigPath: configPath };
	const resolver = RojoResolver.fromPath(configPath);
	const srcDir = fixture("rojocheck", "src");
	const translator = new PathTranslator(srcDir, fixture("rojocheck", "out"), undefined, false, true);
	checkRojoConfig(data, resolver, [srcDir], translator);
	out.rojoCheck = DiagnosticService.flush().map(d => clean(d.messageText));
	checkRojoConfig({ rojoConfigPath: undefined }, resolver, [srcDir], translator);
	out.rojoCheckNoConfig = DiagnosticService.flush().map(d => clean(d.messageText));
}

// fileUsesCommentDirectives
out.comments = {};
for (const name of fs.readdirSync(fixture("comments"))) {
	const text = fs.readFileSync(fixture("comments", name), "utf8");
	const file = ts.createSourceFile(name, text, ts.ScriptTarget.ESNext, true);
	const report = data =>
		fileUsesCommentDirectives(data, file).map(d => ({ message: d.messageText, start: d.start, length: d.length, id: d.id, text: text.slice(d.start, d.start + d.length) }));
	out.comments[name] = report({ projectOptions: options({}) });
	if (report({ projectOptions: options({ allowCommentDirectives: true }) }).length !== 0) throw new Error("allowCommentDirectives");
}

// findAncestorDir and isPathDescendantOf
const native = process.platform === "win32" ? "C:\\" : "/";
const p = (...parts) => native + parts.join(path.sep);
const dirSets = [
	[p("proj", "src")],
	[p("proj", "src"), p("proj", "src", "sub")],
	[p("proj", "src", "a"), p("proj", "src", "b")],
	[p("proj", "src", "a") + path.sep, p("proj", "lib", "b") + path.sep],
	[p("proj", "src", "..", "lib"), p("proj", "lib", "x")],
	[p("proj"), p("other")],
	[p("proj", "ab"), p("proj", "abc")],
];
out.ancestor = dirSets.map(dirs => ({ dirs, result: findAncestorDir(dirs) }));
const descendantCases = [
	[p("a", "b"), p("a")], [p("a"), p("a", "b")], [p("a"), p("a")], [p("a", "..b"), p("a")], [p("a", "b"), p("c")], [p("ab"), p("a")],
];
out.descendant = descendantCases.map(([file, dir]) => ({ file, dir, result: isPathDescendantOf(file, dir) }));

// createPathTranslator on real programs, then the checks at the start of compileFiles
out.translators = {};
for (const name of ["tsconfig.json", "tsconfig.incremental.json", "tsconfig.buildinfo.json", "tsconfig.declaration.json", "tsconfig.rootdirs.json"]) {
	const tsConfigPath = fixture("program", name);
	const data = createProjectData(tsConfigPath, options({ rojo: fixture("program", "rojo-ok.json"), includePath: fixture("program", "include") }));
	const program = createProjectProgram(data);
	const t = createPathTranslator(program, data);
	out.translators[name] = {
		rootDir: clean(t.rootDir),
		outDir: clean(t.outDir),
		buildInfoOutputPath: t.buildInfoOutputPath === undefined ? null : clean(t.buildInfoOutputPath),
		declaration: t.declaration,
		useLuauExtension: t.useLuauExtension,
	};
}

out.compile = {};
for (const [name, project, rojo, include, extra] of [
	["ok", "program", "rojo-ok.json", "include", {}],
	["model", "program", "rojo-model.json", "include", {}],
	["noRojoFile", "program", undefined, "include", {}],
	["noIncludeData", "program", "rojo-ok.json", "elsewhere", {}],
	["serverInclude", "program", "rojo-server.json", "include", {}],
	["clientInclude", "program", "rojo-client.json", "include", {}],
	["isolatedInclude", "program", "rojo-isolated.json", "include", {}],
	["typePackage", "program", undefined, "include", { type: "package" }],
	["inSrc", "program", "rojo-insrc.json", "include", {}],
	["badName", "badname", "rojo-ok.json", "include", {}],
]) {
	const tsConfigPath = fixture(project, "tsconfig.json");
	const rojoPath = rojo ? fixture(project, rojo) : undefined;
	const data = createProjectData(tsConfigPath, options({ ...extra, rojo: rojoPath, includePath: fixture(project, include) }));
	if (!rojo) data.rojoConfigPath = undefined;
	const program = createProjectProgram(data);
	const translator = createPathTranslator(program, data);
	try {
		const result = compileFiles(program.getProgram(), data, translator, []);
		out.compile[name] = { skipped: result.emitSkipped, diagnostics: result.diagnostics.map(d => clean(d.messageText)) };
	} catch (e) {
		// The fixtures have no compiler-types, so reaching MacroManager means the checks before transforming passed.
		if (!e.diagnostics || !e.diagnostics[0].messageText.startsWith("MacroManager could not find")) throw e;
		out.compile[name] = { skipped: false, diagnostics: [] };
	}
}

process.stdout.write(JSON.stringify(out));
