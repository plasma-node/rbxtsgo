// Options whose defaults changed between TS 5.9 and TS 7.
import "@rbxts/not-a-real-side-effect-package";
import * as callable from "./callableExport";
import * as callableNamespace from "./callableNamespace";

print(callable());
print(callableNamespace(), callableNamespace.version);
