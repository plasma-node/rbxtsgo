# Design notes

How rbxtsgo is built, and where it differs from JS `rbxtsc`. The reference compiler is `roblox-ts@3.0.0` (TypeScript 5.5.3). rbxtsgo runs on TypeScript 7.1's Go compiler.

- [The bridge](bridge.md): the one package that touches TypeScript 7 internals, and the rules it enforces
- [Config normalization](config-normalization.md): how a tsconfig written for `rbxtsc`, or for TS 7, is rewritten before checking
- [Setget](setget.md): the built-in port of `rbxts-transformer-setget`, and the fallback to the JS compiler for other plugins
- [Watch mode and incremental builds](watch-and-incremental.md): what gets skipped, and why the output still equals a clean build
- [Union member order](union-order.md): why TS 7's type ordering needed a fix in macro lookups
- [Known differences](known-differences.md): every difference from `rbxtsc` that is accepted, and the ones still open

Output compatibility is a requirement. Every accepted difference is written down with its reason, and expected output is never edited by hand to make a test pass: it comes from running the real JS compiler.
