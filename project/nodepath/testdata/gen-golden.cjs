// Generates golden output from real Node's path module. Run through scripts/gen-goldens.sh.
// Every function is run on both path.win32 and path.posix on a cross product of tricky inputs.
const path = require("path");
const fs = require("fs");

const parts = [
	"", ".", "..", "/", "//", "///", "\\", "\\\\", "a", "a/", "a\\", "/a", "\\a", "a/b", "a\\b", "a/b/", "a/b/c.d.ts",
	"/a/b", "/a/b/", "//a/b", "//a/b/", "//a/b/c", "///a/b", "\\\\server\\share", "\\\\server\\share\\", "\\\\server\\share\\dir\\file.txt",
	"//server/share/dir/file.txt", "\\\\?\\C:\\x", "C:", "C:\\", "C:/", "C:a", "C:\\a", "C:/a", "C:\\a\\b", "c:\\A\\B", "c:\\a\\b\\", "D:\\a", "d:",
	"C:\\Users\\Me\\Project\\src\\index.ts", "c:\\users\\me\\project\\out\\init.lua", "/home/me/project/src/index.ts", "/home/me/project/out/init.lua",
	".gitignore", "a/.gitignore", "file.", "file..", "..file", "...", "a..", "a/..", "a/../b", "a/./b", "./a", "../a", "../../a", "a/b/../../..", "/..", "/../a",
	"foo.server.ts", "foo.client.tsx", "foo.d.ts", "foo.d.tsx", "index.ts", "init.lua", "init.luau", "foo.project.json", "default.project.json",
	"caf\u00e9/\u00e9l\u00e8ve.ts", "\u65e5\u672c\u8a9e/\u30d5\u30a1\u30a4\u30eb.ts", "a b/c d.ts", "C:\\Program Files\\x y\\z.lua",
];
const suffixes = ["", ".ts", ".lua", ".d.ts", "c", "b", "file", ".", "a", "index.ts", "\\", "/"];
const pairsFrom = parts;
const pairsTo = parts;

function cases(flavor) {
	const p = path[flavor];
	const out = { resolve: [], normalize: [], isAbsolute: [], join: [], relative: [], dirname: [], basename: [], extname: [] };
	for (const a of parts) {
		out.normalize.push([a, p.normalize(a)]);
		out.isAbsolute.push([a, p.isAbsolute(a)]);
		out.dirname.push([a, p.dirname(a)]);
		out.extname.push([a, p.extname(a)]);
		for (const s of suffixes) out.basename.push([a, s, p.basename(a, s)]);
	}
	for (const a of parts) {
		for (const b of parts) {
			out.join.push([a, b, p.join(a, b)]);
			// Inputs that depend on the working directory are left out, because the cwd differs between runs.
			if (isRooted(flavor, a) && isRooted(flavor, b)) {
				out.resolve.push([a, b, p.resolve(a, b)]);
			}
		}
	}
	for (const a of pairsFrom) {
		for (const b of pairsTo) {
			if (a === b || (isRooted(flavor, a) && isRooted(flavor, b))) {
				out.relative.push([a, b, p.relative(a, b)]);
			}
		}
	}
	return out;
}

function isRooted(flavor, s) {
	if (flavor === "posix") return s.startsWith("/");
	return /^[A-Za-z]:[\\/]/.test(s) || /^\\\\[^\\]+\\[^\\]+/.test(s) || /^\/\/[^/]+\/[^/]+/.test(s);
}

const out = process.argv[2];
fs.writeFileSync(out, JSON.stringify({ node: process.version, win32: cases("win32"), posix: cases("posix") }));
