import Fallback, { literalExport, Shared, sharedInstance } from "./crossA";
import * as all from "./crossA";

const s = new Shared();
s.data = 1;
s.data++;
print(s.data, sharedInstance.data, literalExport.y);
print(all.sharedInstance.data, new all.Shared().data);
const f = new Fallback();
print(f.extra, f.data);
