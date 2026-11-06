// What the cases do as plain JS, which is what the fixes have to match. Each case's TypeScript is compiled by tsc and run
// by Node, with print and the two Roblox names the cases use. Output has the same shape as run.luau's.
// Run: node oracle.cjs <path to typescript> <the project's src folder>
"use strict";
const fs = require("fs");
const path = require("path");
const Module = require("module");
const ts = require(process.argv[2]);
const src = path.resolve(process.argv[3]);

require.extensions[".ts"] = (module, filename) => {
	const { outputText } = ts.transpileModule(fs.readFileSync(filename, "utf8"), {
		compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true },
	});
	module._compile(outputText, filename);
};
globalThis.print = (...args) => console.log(args.map(String).join(" "));
globalThis.typeIs = (value, name) => (name === "nil" ? value === undefined : typeof value === name);

const cases = path.join(src, "cases");
for (const name of fs.readdirSync(cases).sort()) {
	console.log("== " + path.basename(name, ".ts"));
	Module._load(path.join(cases, name), module);
}
