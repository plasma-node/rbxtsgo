// Package tsbridge is the one place the port touches TypeScript 7 internals where traps exist:
// program creation, config normalization, diagnostics and checker access.
//
// The rules it enforces are explained in docs/bridge.md:
//
//   - One worker per checker. A checker is only ever used by the worker that owns it.
//   - Types and symbols never cross checkers. Each worker has its own macro symbol table.
//   - An emit resolver only comes from Worker.CheckFileAndGetResolver, which checks the file first.
//   - Diagnostics run as a separate phase, before the workers start. The phases must not overlap.
//
// Code outside this package must not call Program.GetTypeCheckerForFileExclusive,
// Program.ForEachCheckerParallel or Checker.NewEmitResolver. TestNoTrapCallsOutsideBridge enforces it.
package tsbridge
