# The bridge

`tsbridge` is the one place rbxtsgo touches TypeScript 7 internals where there are traps: program creation, config normalization, diagnostics and checker access. The rest of the compiler (transformer, Luau AST, project layer) uses it and never calls those APIs directly.

## Rules it enforces

- **One worker per checker.** A checker is only ever used by the worker that owns it.
- **Types and symbols never cross checkers.** Each worker has its own macro symbol table.
- **An emit resolver only comes from `Worker.CheckFileAndGetResolver`, which checks the file first.** TS 7's `NewEmitResolver` does not check the file. `IsReferencedAliasDeclaration` reads links that checking sets, so a resolver made without checking silently gives wrong answers: in the test fixture, 16 of 22 imports came out different. The bridge function checks the file on a checker and then returns that checker's resolver.
- **Diagnostics run as a separate phase, before the workers start.** The phases must not overlap. Calling a resolver method while holding the checker pool's exclusive lock deadlocks, so the bridge function is never called under that lock.
- **No trap calls outside the package.** Code elsewhere must not call `Program.GetTypeCheckerForFileExclusive`, `Program.ForEachCheckerParallel` or `Checker.NewEmitResolver`. `TestNoTrapCallsOutsideBridge` enforces it.

## Diagnostics

The port reports diagnostics the way `compileFiles.ts` does: it checks the source files it is given and reports only for those. It never sets `skipLibCheck`, and errors in packages' declaration files are neither computed nor reported. TS2441 is filtered only when the reported name is `Object` (TS 7 added that check, and it means nothing for Luau output). TS2441 for `exports` and `require` is still reported, because `rbxtsc` reports it too.

## Patches to TypeScript

rbxtsgo builds against a pinned TypeScript 7.1 commit that `scripts/setup-third-party.sh` clones into `third_party/`, then applies the patches in `patches/typescript/`:

- `0001`: exports a few checker functions the port needs and TS 7 does not export.
- `0002`: fixes false TS2802 errors on iterables (`Iterable`, `Map`, `Set`, `Generator`) in the roblox-ts type definitions.
- `0004`: a printer option that keeps the parentheses of the parse tree, off unless a printer asks for it. The setget reprint needs it to match `rbxtsc` byte for byte.

## Checker and AST against TypeScript 5.5.3

`transformer/probe_test.go` compares the Go checker and AST with TypeScript 5.5.3 node by node, on a fixture with `@rbxts` packages and on a from-scratch syntax fixture. It compares symbols, type flags, signature and property counts, constant values and literal text. Everything matches except how some types print (union member order, see [union order](union-order.md)) and one lone surrogate in a string literal. The golden files are made by the real TypeScript 5.5.3 with `transformer/testdata/gen-probe.cjs`, and write the fixture folder as `<ROOT>`.
