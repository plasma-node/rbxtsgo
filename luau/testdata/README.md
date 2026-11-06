# Golden corpus for the luau package

The files in `golden/` are made by the real npm `@roblox-ts/luau-ast@2.0.0` (exact version, see `package.json`). They are the expected output the Go package must match byte for byte. Everything here is generated from public packages.

| File | Content |
|---|---|
| `golden/handwritten.json` | One case per renderer branch, from `cases.js` |
| `golden/random.json` | 160 seeded random programs from `random.js` (shared nodes, temp ids, nested scopes) |
| `golden/scenarios.json` | Object graphs after `create` and `list` operations, from `scenarios.js` |
| `golden/util.json` | Predicates, kind names, kind guards, JS number conversion, `number()` |

A render case holds the AST as a graph (`graph.js`), the render mode, and either `expected` (the exact Luau text) or `error` (the assertion message upstream throws), plus the temp names `solveTempIds` chose. The graph keeps object identity, parent pointers, shared nodes and raw list-node links, so Go renders the same structure JS did. Go tests: `golden_test.go`, `scenarios_test.go`, `util_test.go`, `jsnum_test.go`, `api_test.go`.

## Regenerate

```sh
cd luau/testdata
npm ci            # installs @roblox-ts/luau-ast 2.0.0 into node_modules (ignored by git)
node generate.js          # rewrite golden/
node generate.js --check  # fail if golden/ is out of date
```

Output is deterministic. `golden/*` is marked `-text` in `.gitattributes`, so line endings never change. When you add a case to `cases.js` or a scenario to `scenarios.js`, regenerate; scenarios also need the same steps added to `scenarios_test.go`.

## Extra goldens (for example from a patched upstream compiler)

`graph.js` exports `serialize({ ast: list })`. A patched `rbxtsc` can write `{ "mode": "renderAST", "graph": serialize({ast}), "expected": renderAST(list), "temps": {} }` cases into a file of the same shape as `handwritten.json` (`{ "luauAstVersion": "2.0.0", "cases": [...] }`). Point the Go tests at the folder:

```sh
LUAU_EXTRA_GOLDEN=/path/to/dir go test ./luau -run TestGoldenExtra
```

The test is skipped when the variable is unset, so nothing depends on it. Upstream roblox-ts has no checked-in Luau snapshots (its tests compile TypeScript and run it in Luau), so a compiler-level differential belongs to the transformer phase.
