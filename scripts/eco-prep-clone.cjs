// Purpose: for validate-ecosystem.sh. Rewrites a cloned project's package.json so that npm installs only the
//          Roblox-side packages (@rbxts/*, @flamework/*, rbxts-transformer-flamework, @rbxts-js/*), never lint or build
//          tooling and never a "file:" or "link:" dependency. The original is kept as package.json.orig. pnpm catalog:
//          specs are resolved from pnpm-workspace.yaml. Prints what it dropped and whether the tsconfig lists plugins.
// Run:     node scripts/eco-prep-clone.cjs <project dir>
// Studio:  not needed.
// Date:    2026-11-05, ecosystem validation.
const fs = require("fs");
const path = require("path");
const dir = process.argv[2];
const pkgPath = path.join(dir, "package.json");
if (!fs.existsSync(pkgPath + ".orig")) fs.copyFileSync(pkgPath, pkgPath + ".orig");
const pkg = JSON.parse(fs.readFileSync(pkgPath + ".orig", "utf8"));

// A small reader for the catalog blocks of pnpm-workspace.yaml: "catalog:" and "catalogs:" with named children.
const catalogs = {};
const wsPath = path.join(dir, "pnpm-workspace.yaml");
if (fs.existsSync(wsPath)) {
	let current = null;
	let inCatalogs = false;
	for (const raw of fs.readFileSync(wsPath, "utf8").split(/\r?\n/)) {
		if (/^\s*#/.test(raw) || raw.trim() === "") continue;
		const indent = raw.match(/^ */)[0].length;
		const line = raw.trim();
		if (indent === 0) {
			inCatalogs = line === "catalogs:";
			current = line === "catalog:" ? (catalogs.default = {}) : null;
			continue;
		}
		if (inCatalogs && indent <= 2 && line.endsWith(":")) {
			current = catalogs[line.slice(0, -1)] = {};
			continue;
		}
		const m = line.match(/^["']?([^"':]+(?::[^"':]+)?)["']?\s*:\s*["']?([^"']+)["']?$/);
		if (current && m) current[m[1]] = m[2];
	}
	// "catalog:prod" reads the catalog named prod; plain "catalog:" reads the default one.
}

// pnpm-lock.yaml (v9): an importer entry is a name line, then a specifier line, then a version line. Pin each Roblox
// package to the version pnpm chose, so that npm does not pick newer typings than the project was written against.
const locked = {};
const lockPath = path.join(dir, "pnpm-lock.yaml");
if (fs.existsSync(lockPath)) {
	const lines = fs.readFileSync(lockPath, "utf8").split(/\r?\n/);
	for (let i = 0; i + 2 < lines.length; i++) {
		const m = lines[i].match(/^ {4,6}['"]?([^'":]+(?::[^'":]+)?)['"]?:$/);
		if (!m || !/^\s+specifier:/.test(lines[i + 1])) continue;
		const v = lines[i + 2].match(/^\s+version:\s*(\S+)/);
		if (v && locked[m[1]] === undefined) locked[m[1]] = v[1].replace(/\(.*$/, "");
	}
}
const keep = (name) => /^@rbxts\//.test(name) || /^@flamework\//.test(name) || /^@rbxts-js\//.test(name) || name === "rbxts-transformer-flamework";
const out = {};
const dropped = [];
for (const section of ["dependencies", "devDependencies", "peerDependencies"]) {
	for (const [name, spec] of Object.entries(pkg[section] || {})) {
		if (!keep(name)) {
			if (section !== "devDependencies") dropped.push(`${name} (not a Roblox package)`);
			continue;
		}
		let version = spec;
		if (version.startsWith("catalog:")) {
			const cat = catalogs[version.slice(8) || "default"];
			version = cat && cat[name];
			if (!version) {
				dropped.push(`${name} (unresolved ${spec})`);
				continue;
			}
		}
		if (/^(file|link|workspace|git|github|http)/.test(version)) {
			dropped.push(`${name}@${version}`);
			continue;
		}
		if (locked[name] && /^[0-9]/.test(locked[name])) version = locked[name];
		out[name] = version;
	}
}
const tsconfig = path.join(dir, "tsconfig.json");
const text = fs.existsSync(tsconfig) ? fs.readFileSync(tsconfig, "utf8") : "";
const plugins = [...text.matchAll(/"transform"\s*:\s*"([^"]+)"/g)].map((m) => m[1]);
const flamework = plugins.includes("rbxts-transformer-flamework");
if (flamework) {
	out["roblox-ts"] = "3.0.0";
	out["typescript"] = "5.5.3";
	if (!out["rbxts-transformer-flamework"]) out["rbxts-transformer-flamework"] = "1.3.2";
}
fs.writeFileSync(pkgPath, JSON.stringify({ name: "eco-clone", version: "0.0.0", private: true, dependencies: out }, null, "\t") + "\n");
console.log(JSON.stringify({ kept: Object.keys(out).length, dropped, plugins }));
