// Generates golden output from the real @roblox-ts/rojo-resolver@1.1.0, on the real filesystem, so the result is
// per-platform. Run through scripts/gen-goldens.sh on each platform.
// Usage: node gen-golden.cjs <node_modules with the package>
const fs = require("fs");
const path = require("path");
const modules = path.resolve(process.argv[2]);
const { RojoResolver, RbxPathParent } = require(path.join(modules, "@roblox-ts/rojo-resolver"));

const root = path.join(__dirname, "fixtures");
const cases = JSON.parse(fs.readFileSync(path.join(__dirname, "cases.json"), "utf8"));
const clean = text => text.split(root).join("<ROOT>");
const abs = rel => path.join(root, ...rel.split("/"));
const show = rbxPath => (rbxPath === undefined ? "undefined" : JSON.stringify(rbxPath));

function describe(resolver) {
	const lines = [];
	for (const warning of resolver.getWarnings()) lines.push("warning " + clean(warning));
	lines.push("isGame " + resolver.isGame);
	for (const partition of resolver.getPartitions()) {
		lines.push(`partition ${clean(partition.fsPath)} ${JSON.stringify(partition.rbxPath)}`);
	}
	const seen = [];
	for (const rel of cases.lookups) {
		const rbxPath = resolver.getRbxPathFromFilePath(abs(rel));
		lines.push(`lookup ${rel} ${show(rbxPath)}`);
		if (rbxPath !== undefined) {
			lines.push(`  isolated ${resolver.isIsolated(rbxPath)} network ${resolver.getNetworkType(rbxPath)}`);
			if (!seen.some(v => JSON.stringify(v) === JSON.stringify(rbxPath))) seen.push(rbxPath);
		}
	}
	for (const from of seen) {
		for (const to of seen) {
			const relative = RojoResolver.relative(from, to).map(v => (v === RbxPathParent ? "<parent>" : v));
			lines.push(`pair ${JSON.stringify(from)} ${JSON.stringify(to)} ${resolver.getFileRelation(from, to)} ${JSON.stringify(relative)}`);
		}
	}
	return lines;
}

function attempt(fn) {
	try {
		return { lines: describe(fn()) };
	} catch {
		return { throws: true };
	}
}

const out = { node: process.version, fromPath: {}, synthetic: {}, invalid: {}, find: {}, types: {}, relative: {} };
for (const config of cases.configs) out.fromPath[config] = attempt(() => RojoResolver.fromPath(abs(config)));
for (const dir of cases.synthetic) out.synthetic[dir] = attempt(() => RojoResolver.synthetic(abs(dir)));
for (const name of fs.readdirSync(path.join(root, "invalid"))) {
	out.invalid[name] = attempt(() => RojoResolver.fromPath(path.join(root, "invalid", name)));
}
for (const dir of cases.find) {
	const { path: found, warnings } = RojoResolver.findRojoConfigFilePath(abs(dir));
	out.find[dir] = { path: found === undefined ? null : clean(found), warnings: warnings.map(clean) };
}
const typeResolver = RojoResolver.synthetic(abs("model/out"));
for (const name of cases.types) out.types[name] = typeResolver.getRbxTypeFromFilePath(name);
const rbx = [[], ["a"], ["a", "b"], ["a", "c"], ["a", "b", "c"], ["d"], ["ReplicatedStorage", "x"], ["StarterGui", "x"], ["StarterPlayer", "StarterPlayerScripts", "y"]];
for (const from of rbx) {
	for (const to of rbx) {
		out.relative[JSON.stringify(from) + JSON.stringify(to)] = RojoResolver.relative(from, to).map(v => (v === RbxPathParent ? "<parent>" : v));
	}
}
const flavor = process.platform === "win32" ? "windows" : process.platform;
fs.writeFileSync(path.join(__dirname, `golden.${flavor}.json`), JSON.stringify(out, null, "\t") + "\n");
