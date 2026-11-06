// Command genbench writes a synthetic roblox-ts project for timing watch mode: public code only, with no resemblance to
// anything real on purpose. Usage: genbench <dir> <modules> [setget]. With setget, the classes have accessors and the
// tsconfig turns the built-in transform on. The folder needs node_modules/@rbxts (types, compiler-types,
// services) to build; scripts/bench-watch.sh --generate puts it there.
//
// Modules form layers. Each imports up to four modules from the layers below, so a change low down reaches many files
// and a change high up reaches few, like a real project. The shapes are what lowering spends time on: classes, array and
// string macros, Maps, enums, destructuring, optional chains, Instance APIs and exports.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/microsoft/TypeScript/tsc/rbxtsgo/plugins"
)

const tsconfig = `{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"jsx": "react",
		"jsxFactory": "Roact.jsx",
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"resolveJsonModule": true,
		"forceConsistentCasingInFileNames": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"experimentalDecorators": true,
		"rootDir": "src",
		"outDir": "out",
		"baseUrl": "src"
	},
	"include": ["src/**/*"]
}
`

const rojo = `{
	"name": "bench",
	"globIgnorePaths": ["**/package.json", "**/tsconfig.json"],
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"rbxts_include": {
				"$path": "include",
				"node_modules": {
					"$className": "Folder",
					"@rbxts": { "$path": "node_modules/@rbxts" }
				}
			},
			"TS": { "$path": "out/shared" }
		},
		"ServerScriptService": {
			"$className": "ServerScriptService",
			"TS": { "$path": "out/server" }
		}
	}
}
`

func module(rng *rand.Rand, index int, imports []int, setget bool) string {
	var b strings.Builder
	name := "M" + strconv.Itoa(index)
	for _, dep := range imports {
		boxImport := ""
		if setget {
			boxImport = fmt.Sprintf(", box%d", dep)
		}
		fmt.Fprintf(&b, "import { M%dService, M%dKind, M%dConfig, make%d%s } from \"shared/l%d/m%d\";\n", dep, dep, dep, dep, boxImport, dep%7, dep)
	}
	body := `
export const enum %[1]sKind {
	Alpha = %[2]d,
	Beta,
	Gamma,
}

export interface %[1]sConfig {
	readonly name: string;
	size: number;
	tags: Array<string>;
	child?: %[1]sConfig;
	apply(value: number): number;
	callback: (value: number) => number;
}

export function make%[2]d(name: string, size = %[2]d): %[1]sConfig {
	return {
		name,
		size,
		tags: [name, tostring(size)],
		apply(value) {
			return value + size;
		},
		callback: value => value * size,
	};
}

export class %[1]sService {
	private readonly items = new Map<string, %[1]sConfig>();
	private total = 0;
	public static instances = 0;

	public constructor(private readonly label: string) {
		%[1]sService.instances++;
	}

	public add(config: %[1]sConfig) {
		this.items.set(config.name, config);
		this.total += config.apply(config.size) + config.callback(1);
		return this;
	}

	public names() {
		const result = new Array<string>();
		for (const [name, config] of this.items) {
			if (config.child?.child !== undefined || config.size > %[2]d) {
				result.push(name.upper() + this.label);
			}
		}
		result.sort((a, b) => a < b);
		return result.map(value => value.sub(1, 3)).filter(value => value.size() > 0);
	}

	public describe(kind: %[1]sKind) {
		const { total, label } = this;
		switch (kind) {
			case %[1]sKind.Alpha:
				return ` + "`${label}: ${total}`" + `;
			case %[1]sKind.Beta:
				return label.rep(2);
			default:
				return tostring(total + %[1]sKind.Gamma);
		}
	}

	public spawnPart(parent: Instance) {
		const part = new Instance("Part");
		part.Name = this.label;
		part.Size = new Vector3(%[2]d, 1, 1);
		part.Anchored = this.total > 0;
		part.Parent = parent;
		part.Touched.Connect(other => print(other.Name, this.total));
		return part;
	}
}
`
	fmt.Fprintf(&b, body, name, index)
	if setget {
		fmt.Fprintf(&b, `
export class M%[1]dBox {
	private stored = %[1]d;
	public get value() {
		return this.stored;
	}
	public set value(amount: number) {
		this.stored = amount;
	}
}
export const box%[1]d = new M%[1]dBox();
`, index)
		for _, dep := range imports {
			fmt.Fprintf(&b, "export function bump%d_%d() {\n\tbox%d.value += 1;\n\treturn box%d.value;\n}\n", index, dep, dep, dep)
		}
	}
	for i, dep := range imports {
		fmt.Fprintf(&b, `
export function use%d_%d(service: M%dService, config: M%dConfig) {
	const made = make%d(config.name, config.size + %d);
	const [first, second = "fallback"] = made.tags;
	service.add(made).add(config);
	const kinds = [M%dKind.Alpha, M%dKind.Beta, M%dKind.Gamma];
	let sum = 0;
	for (const kind of kinds) {
		sum += kind;
	}
	return { first, second, sum, text: service.describe(M%dKind.Beta), value: config.apply(sum) };
}
`, index, i, dep, dep, dep, rng.IntN(100), dep, dep, dep, dep)
	}
	return b.String()
}

func main() {
	if len(os.Args) != 3 && !(len(os.Args) == 4 && os.Args[3] == "setget") {
		fmt.Fprintln(os.Stderr, "usage: genbench <dir> <modules> [setget]")
		os.Exit(2)
	}
	setget := len(os.Args) == 4
	dir := os.Args[1]
	count, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	rng := rand.New(rand.NewPCG(1, uint64(count)))
	write := func(rel, text string) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o777); err != nil {
			panic(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o666); err != nil {
			panic(err)
		}
	}
	config := tsconfig
	if setget {
		config = strings.Replace(config, "{\n", "{\n\t\""+plugins.BuiltinBlock+"\": { \"setget\": true },\n", 1)
	}
	write("tsconfig.json", config)
	write("default.project.json", rojo)
	write("package.json", "{ \"name\": \"bench\", \"version\": \"0.0.0\", \"private\": true }\n")

	var entry strings.Builder
	for i := range count {
		var imports []int
		if i > 0 {
			seen := map[int]bool{}
			for range min(i, 1+rng.IntN(4)) {
				// Mostly near neighbours, sometimes something low down, as a shared utility would be.
				dep := i - 1 - rng.IntN(min(i, 20))
				if rng.IntN(4) == 0 {
					dep = rng.IntN(min(i, 10))
				}
				if !seen[dep] {
					seen[dep] = true
					imports = append(imports, dep)
				}
			}
		}
		write(fmt.Sprintf("src/shared/l%d/m%d.ts", i%7, i), module(rng, i, imports, setget))
		if i%50 == 0 {
			fmt.Fprintf(&entry, "import { M%dService } from \"shared/l%d/m%d\";\nprint(new M%dService(\"root\").names());\n", i, i%7, i, i)
		}
	}
	write("src/server/main.server.ts", entry.String())
}
