import Klass = require("./exportEqualsClass");
import fn = require("./exportEqualsFn");
import object = require("./exportEqualsObject");
import Space = require("./exportEqualsNamespace");
import nativeFn = require("./nativeFn");
import NativeClass = require("./nativeClass");
import defaulted = require("./defaultExport");
import unused = require("./exportEqualsFn");

export import Again = require("./exportEqualsObject");

print(new Klass().v, fn(), object.a, Space.run(), nativeFn(), new NativeClass().v, defaulted.other, Again.b);
type T = typeof unused;
declare const t: T;
print(t);
