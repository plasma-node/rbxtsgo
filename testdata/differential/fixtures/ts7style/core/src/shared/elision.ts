import { OnlyType, value } from "./viaPaths";
import { other } from "./defaultExport";
import type { default as Named } from "./defaultExport";
import { type OnlyType as Again } from "./viaPaths";
import fn from "./exportEqualsFn";
import Klass from "./exportEqualsClass";
import "./effect";

const a: OnlyType = 1;
const b: Again = 2;
const c: Klass | undefined = undefined;
const d: typeof fn | undefined = undefined;
const e: typeof other | undefined = undefined;
declare const f: typeof Named;
print(a, b, c, d, e, f, value);
