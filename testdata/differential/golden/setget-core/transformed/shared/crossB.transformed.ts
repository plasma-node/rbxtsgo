import Fallback, { literalExport, Shared, sharedInstance } from "./crossA";
import * as all from "./crossA";
const s = new Shared();
s.__setdata(1);
s.__setdata(s.__getdata() + 1);
print(s.__getdata(), sharedInstance.__getdata(), literalExport.__gety());
print(all.sharedInstance.__getdata(), new all.Shared().__getdata());
const f = new Fallback();
print(f.__getextra(), f.__getdata());
