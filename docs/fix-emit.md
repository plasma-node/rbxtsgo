# `--fix-emit`

rbxtsgo stays byte-identical to JS `roblox-ts` 3.0.0 for all code that compiles and runs correctly there. Fixes fall into three classes.

| Class | What it covers | Why |
|---|---|---|
| **On by default** | JS crashes, or JS writes Luau that fails to parse | No working output can change |
| **Behind `--fix-emit`** or `"rbxtsgo": { "fixEmit": true }` | JS writes Luau that parses and does the wrong thing | A project could depend on it, or could see a diff it didn't ask for |
| **Not included** | Anything that changes calling conventions | It would break compatibility with published `@rbxts/*` packages |

Whether output "fails to parse" is measured with Lune's Luau compiler. Upstream's `#3033` landed in both classes for this reason: its issue is true of a template but not of a plain string. See `testdata/fixemit/strings/expected.txt`.

## What `--fix-emit` changes

- A `continue` inside a `switch` in a loop, including through `try`.
- `export { default as X } from "m"` where `m` uses `export =`.
- Functions declared after a `return`, `break` or `continue`.
- Strings that parse but have the wrong value.

## Choices

- **A literal that failed to parse gets the whole string fix,** including the parts that are otherwise flag-only. It could never run before, and half a fix would turn a parse error into a wrong value.
- **Upstream's quote preservation is left out,** because it changes every single-quoted string. With the flag, the source's quote is used only for a string with both quote characters.
- **A new diagnostic takes an id outside JS's order,** because ids are positions in JS's list.
- **`--fix-emit` with a plugin only JS can run is an error,** because handing the build to JS would silently lose the fixes.
- **A `continue` works its way out one level at a time,** from the innermost of switch, try and loop.

## Testing

See [testdata/fixemit/README.md](../testdata/fixemit/README.md). Each `--fix-emit` case runs under Lune and is compared with Node running the same TypeScript. Changing the option forces a rebuild.
