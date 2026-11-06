// Upstream PR #3115's runtime tests, in tests/src/tests/moduleSyntax.spec.ts, printing what they compare. Issue #2799.
// Its string-literal export names are left out, because TS 5.5.3, which JS 3.0.0 runs on, can't parse them.
import { DefaultClass as ExportedDefaultClass, callback, Class, ClassAlias, objectValue } from "../lib/defaultReExport";
import defaultReExport from "../lib/defaultReExport";
import { value as chainedReExport } from "../lib/defaultReExportChain";
import DefaultClass from "../lib/defaultClass";
import ExportEqualsClass = require("../lib/exportEqualsClass");
import exportEqualsFunction = require("../lib/exportEqualsFunction");
import exportEqualsObject = require("../lib/exportEqualsObject");

print("class", Class === ExportEqualsClass);
print("class alias", ClassAlias === ExportEqualsClass);
print("function", callback === exportEqualsFunction);
print("object", defaultReExport === exportEqualsObject);
print("chain", chainedReExport === exportEqualsObject);
print("named property", objectValue);
print("ES default", ExportedDefaultClass === DefaultClass);
print("ES default value", new ExportedDefaultClass().value);
print("called", callback?.());
print("constructed", Class !== undefined ? new Class().value : "nothing");

export {};
