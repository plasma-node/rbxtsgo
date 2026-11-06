import * as object from "./exportEqualsObject";
import * as Space from "./exportEqualsNamespace";
import * as star from "./defaultExport";
import * as asDefault from "./defaultAsName";
import * as nativeDefault from "./nativeDefault";

print(object.a, Space.run(), star.default(), star.other, asDefault.alias, nativeDefault.default);
