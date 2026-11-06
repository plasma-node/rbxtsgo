// Generates golden output from the real @roblox-ts/path-translator@1.1.0. Run through scripts/gen-goldens.sh.
// Usage: node gen-golden.cjs <node_modules with the package> <out file>
//
// The package requires Node's "path". The hook below swaps in path.win32 or path.posix, so one machine can
// produce both flavors.
const Module = require("module");
const fs = require("fs");
const realPath = require("path");
const [modules, outFile] = process.argv.slice(2);

function loadWith(flavor) {
	const pkgFile = require.resolve(realPath.resolve(modules, "@roblox-ts/path-translator"));
	for (const key of Object.keys(require.cache)) {
		if (key.includes("path-translator")) delete require.cache[key];
	}
	const original = Module._load;
	Module._load = function (request, ...rest) {
		if (request === "path") return realPath[flavor];
		return original.call(this, request, ...rest);
	};
	try {
		return require(pkgFile).PathTranslator;
	} finally {
		Module._load = original;
	}
}

const FLAVORS = {
	win32: {
		roots: [
			["C:\\proj\\src", "C:\\proj\\out"],
			["C:\\proj\\src\\", "C:\\proj\\out\\"],
			["C:\\proj\\", "C:\\proj\\out\\"],
			["c:\\Proj\\Src", "C:\\proj\\out"],
			["C:\\", "D:\\out"],
		],
		files: [
			"C:\\proj\\src\\foo.ts", "C:\\proj\\src\\foo.tsx", "C:\\proj\\src\\foo.d.ts", "C:\\proj\\src\\foo.d.tsx",
			"C:\\proj\\src\\index.ts", "C:\\proj\\src\\index.tsx", "C:\\proj\\src\\index.d.ts", "C:\\proj\\src\\index.server.ts",
			"C:\\proj\\src\\index.client.tsx", "C:\\proj\\src\\init.ts", "C:\\proj\\src\\init.server.ts", "C:\\proj\\src\\foo.server.ts",
			"C:\\proj\\src\\foo.client.tsx", "C:\\proj\\src\\foo.transformed.ts", "C:\\proj\\src\\a.b.c.ts", "C:\\proj\\src\\noext",
			"C:\\proj\\src\\.hidden", "C:\\proj\\src\\.hidden.ts", "C:\\proj\\src\\foo.json", "C:\\proj\\src\\foo.lua", "C:\\proj\\src\\foo.luau",
			"C:\\proj\\src\\foo..ts", "C:\\proj\\src\\foo.", "C:\\proj\\src\\sub\\dir\\deep.ts", "C:\\proj\\src\\sub\\..\\up.ts",
			"C:\\proj\\src\\sub\\", "C:\\proj\\src\\sub\\index.ts\\", "C:/proj/src/forward.ts", "c:\\PROJ\\SRC\\Case.ts", "C:\\other\\x.ts", "D:\\x.ts", "C:\\foo.ts", "C:\\",
			"C:\\proj\\out\\foo.lua", "C:\\proj\\out\\foo.luau", "C:\\proj\\out\\init.lua", "C:\\proj\\out\\init.luau", "C:\\proj\\out\\init.server.lua",
			"C:\\proj\\out\\init.client.luau", "C:\\proj\\out\\index.lua", "C:\\proj\\out\\index.server.lua", "C:\\proj\\out\\foo.server.lua",
			"C:\\proj\\out\\foo.d.ts", "C:\\proj\\out\\foo.d.tsx", "C:\\proj\\out\\index.d.ts", "C:\\proj\\out\\foo.txt", "C:\\proj\\out\\sub\\init.lua",
			"\\\\server\\share\\proj\\foo.ts",
		],
	},
	posix: {
		roots: [
			["/proj/src", "/proj/out"],
			["/proj/src/", "/proj/out/"],
			["/proj/", "/proj/out/"],
			["/Proj/Src", "/proj/out"],
			["/", "/out"],
		],
		files: [
			"/proj/src/foo.ts", "/proj/src/foo.tsx", "/proj/src/foo.d.ts", "/proj/src/foo.d.tsx",
			"/proj/src/index.ts", "/proj/src/index.tsx", "/proj/src/index.d.ts", "/proj/src/index.server.ts",
			"/proj/src/index.client.tsx", "/proj/src/init.ts", "/proj/src/init.server.ts", "/proj/src/foo.server.ts",
			"/proj/src/foo.client.tsx", "/proj/src/foo.transformed.ts", "/proj/src/a.b.c.ts", "/proj/src/noext",
			"/proj/src/.hidden", "/proj/src/.hidden.ts", "/proj/src/foo.json", "/proj/src/foo.lua", "/proj/src/foo.luau",
			"/proj/src/foo..ts", "/proj/src/foo.", "/proj/src/sub/dir/deep.ts", "/proj/src/sub/../up.ts",
			"/proj/src/sub/", "/proj/src/sub/index.ts/", "/PROJ/SRC/Case.ts", "/other/x.ts", "/foo.ts", "/", "//double/foo.ts",
			"/proj/out/foo.lua", "/proj/out/foo.luau", "/proj/out/init.lua", "/proj/out/init.luau", "/proj/out/init.server.lua",
			"/proj/out/init.client.luau", "/proj/out/index.lua", "/proj/out/index.server.lua", "/proj/out/foo.server.lua",
			"/proj/out/foo.d.ts", "/proj/out/foo.d.tsx", "/proj/out/index.d.ts", "/proj/out/foo.txt", "/proj/out/sub/init.lua",
		],
	},
};

const result = { node: process.version };
for (const [flavor, { roots, files }] of Object.entries(FLAVORS)) {
	const PathTranslator = loadWith(flavor);
	const rows = [];
	for (const [rootDir, outDir] of roots) {
		for (const declaration of [false, true]) {
			for (const luau of [false, true]) {
				const t = new PathTranslator(rootDir, outDir, undefined, declaration, luau);
				for (const file of files) {
					rows.push({
						rootDir, outDir, declaration, luau, file,
						output: t.getOutputPath(file),
						declarationPath: t.getOutputDeclarationPath(file),
						transformed: t.getOutputTransformedPath(file),
						inputs: t.getInputPaths(file),
						importPath: t.getImportPath(file, false),
						importPathNodeModule: t.getImportPath(file, true),
					});
				}
			}
		}
	}
	result[flavor] = rows;
}
fs.writeFileSync(outFile, JSON.stringify(result));
