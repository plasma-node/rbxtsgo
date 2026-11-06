# Config normalization

TS 7 removed several options that `rbxtsc` configs use. `tsbridge.NormalizeOptions` rewrites them after parsing and before the program is created, so a tsconfig that works with `rbxtsc` works unchanged. Running it twice changes nothing the second time, and verbose output lists each change.

| Input | Rewritten to | Why |
|---|---|---|
| `module: preserve` with `moduleResolution: bundler` | `commonjs` with `node10`, then the rows below apply | See below |
| `baseUrl` | `paths` with `"*": ["./*"]` added, based at the old `baseUrl`; `baseUrl` cleared | TS 7 removed `baseUrl`. Before it, `paths` were relative to `baseUrl` |
| `moduleResolution: node10` | `bundler` | TS 7 removed `node10` as a setting |
| ... with `resolvePackageJsonExports` / `resolvePackageJsonImports` unset | `false` | `node10` ignores those `package.json` fields, so deep imports only resolve the same way with them off. An explicit value is respected |
| `downlevelIteration` | dropped | Removed in TS 7 |
| `types` unset | `["*"]` | TS 5 included every package under `typeRoots` by default. TS 7 includes none |
| `noUncheckedSideEffectImports` unset | `false` | The default changed in TS 6 and adds TS2882 errors that `rbxtsc` does not report |

A classic `module` such as `commonjs` is left alone.

## TS 7 style configs

TS 7 users write `module: preserve` and `moduleResolution: bundler`, with `paths` and an explicit `types`. `rbxtsc` rejects that pair. rbxtsgo accepts exactly that pair and compiles it as the classic `commonjs` and `node10`. Any other combination, such as `commonjs` with `bundler`, gets the same error as before.

Left as written, Luau was the same as `rbxtsc` for `export =`, default imports, re-exports, import elision, JSON, dynamic `import()` and `paths`, but two things differed. A package whose `exports` disagrees with its `main` resolved to a different file, because bundler resolution honors `exports`, and top-level `await` compiled where `rbxtsc` rejects it with TS1378. Rewriting the pair removes both differences, so only one configuration reaches the checker and there is no second behavior to maintain.

Consequences:

- An editor running TS 7 can disagree with the build, because it honors `exports` and allows top-level `await`. The build matches `rbxtsc`.
- TS1378's message can read oddly: TS 7 lists `preserve` among the allowed settings.
- An explicit `types` list is the user's own. Packages outside it do not load their globals, as with TS 7.
- `rootDir` defaults to the tsconfig's folder since TS 6, where TS 5 used the common folder of the emitted files. For a project that sets only `rootDirs`, `Project.CommonSourceDirectory` computes the TS 5 value, and diagnostics TS5011 and TS6059 are dropped when `rootDir` is unset.

## Project references

`references` are ignored, as `rbxtsc` ignores them. JS builds the program without project references, and reports TS6059 for a file imported across one. Given the references, TS 7 would report TS6305 instead, so `tsbridge.NewProject` drops them.
