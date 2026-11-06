# Union member order and macro lookups

## The difference

Upstream's `getFirstDefinedSymbol` returns the symbol of the first member of `type.types` that has one. It decides macro lookups in call expressions, optional chains, `for` and `for...of`, `return`, object and JSX spreads, index diagnostics and method-index checks.

- TypeScript 5.5.3 sorts a union by type id, which is creation order.
- TypeScript 7 sorts by a deterministic comparison: flags, then names, then declaration position, where declarations in different files compare by the file's index in the program.

Go's order does not vary between runs or with the number of checkers. It does depend on where a file falls in the program.

## It changes output

The fixture `testdata/differential/fixtures/core/src/shared/union_macros.ts` writes each case in both member orders:

```ts
declare const macroOrCustom: Array<string> | HasSize;
export const a = macroOrCustom.size();
```

`rbxtsc` writes `#macroOrCustom` in either member order. TS 7's order gave `macroOrCustom:size()`, a real difference, because `HasSize.size` and `Array.size` are different calls at runtime. In `rbxtsc` the macro wins whichever way round the source writes the union.

## Why JS behaves that way

`compileFiles` builds `MacroManager` before checking any file. Its constructor calls `getTypeAtLocation` on every method signature of every macro class, in `PROPERTY_CALL_MACROS` order. So each of those methods has a type with a low id before any user code is checked, and in any union those members sort first, in that order. A probe of TypeScript 5.5.3 supports this: type ids in a test union were around 88 without `MacroManager` and around 231 with it.

## The fix

`NewMacroSymbols` records the order in which it visits method symbols, which is the same order. For a union only, `getFirstDefinedSymbol` returns the member with the lowest recorded order if any member has one, and otherwise the first member with a symbol, as before. Intersections keep their written order on both sides.

That reproduces TypeScript 5.5.3's choice for property call macros without needing type ids. Reproducing type ids was rejected because they follow check order on one checker, so they would change with the number of workers, and output must not. Forcing a single checker was rejected because it gives up the parallel speedup.

## What is fitted rather than derived

Call macros in unions can only be written through a type, as in `declare const f: ((v: unknown) => void) | typeof assert`. `rbxtsc` writes `assert(true)` for that in both member orders, but a direct probe predicts no macro, so something in the compile path changes the answer and it was not found. The port ranks call macros first, right after macro class methods, because that matches every case measured. This part is fitted to observations. It is covered by the `union-isolated` corpus and `union_macros.ts`, and is a rare pattern.

## Also order-sensitive, not changed

- `isMethodFromType` walks union members with a short circuit, so which member's check runs can differ, and with it whether `noMixedTypeCall` is reported for a union mixing methods and callbacks. No corpus case shows a difference.
- `validateSpread` reads `getProperties`, whose order on spread types differs, which can reorder diagnostics. No corpus case shows a difference.
- TypeScript's own diagnostic text prints unions in the new order.

Setting `RBXTSGO_TRACE_UNION=<file>` logs each lookup on a union or intersection whose members have two or more distinct symbols.
