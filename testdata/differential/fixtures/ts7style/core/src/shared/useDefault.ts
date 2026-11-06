import Klass from "./exportEqualsClass";
import fn from "./exportEqualsFn";
import object from "./exportEqualsObject";
import Space from "./exportEqualsNamespace";
import nativeFn from "./nativeFn";
import NativeClass from "./nativeClass";
import nativeDefault from "./nativeDefault";
import named, { other } from "./defaultExport";
import asDefault, { alias } from "./defaultAsName";

print(new Klass().v, fn(), object.a, Space.run(), nativeFn(), new NativeClass().v, nativeDefault);
print(named(), other, asDefault, alias);
