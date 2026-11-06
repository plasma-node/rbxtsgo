class Holder {
    private inner = 1;
    public __getv(): number {
        return this.inner;
    }
    public __setv(val: number) {
        this.inner = val;
    }
    public __getval(): Holder | undefined {
        return this;
    }
    public __setval(value: Holder | undefined) {
        print(value);
    }
    public field = 1;
    public maybe?: Holder;
}
const h = new Holder();
const maybe = h as Holder | undefined;
// optional chaining
print(maybe?.__getv?.());
print(maybe?.__getval?.()?.__getv?.());
print(h.__getval()?.__getv?.());
print(h.maybe?.__getval?.()?.field);
print(h.maybe?.field);
// non-null, parens and casts
print((h.__getval())!.__getv());
(h.__getval())!.__setv(2);
(h.__getval())!.__setv((h.__getval())!.__getv() + 2);
(h).__setv(3);
h.__setv(4);
(h as Holder).__setv(5);
h.__setv(6);
print((h as Holder).__getv());
print((h).__getv());
// not a set when it's only under the assignment's right side
const arr = [1, 2];
h.__setv(1);
let plain = 0;
plain = h.field;
print(plain);
const obj = { key: 1 };
obj.key = h.field;
obj.key += h.field;
print(obj);
// the first property access anywhere under the expression decides
h.__setv(h.__getv());
h.__setv(h.__getv() + h.__getv());
print(plain);
h.__setv(h.__getv());
const y = (h.__setv(h.__getv()));
print(y);
let z = h.__getv() > 1 ? h.__getv() : 0;
h.__setv(h.__getv() > 1 ? h.field : 0);
print(z);
print(h.__getv() + h.__getv(), h.__getv() === 1, h.__getv() !== h.field);
h.__setv(h.__getv() + 1);
h.__setv(h.field);
h.__setval(h);
h.__setval(h.__getval());
h.field = h.__getv();
const nested = (h.__setv(1));
print(nested);
// calls and declarations
function take(a: number, b?: number) {
    return a + (b ?? 0);
}
print(take(h.__getv()), take(h.__getv(), h.__getv()));
const fn = () => h.__getv();
print(fn());
const typed: number = h.__getv();
print(typed);
const [first] = [h.__getv()];
const { key: renamed } = { key: h.__getv() };
print(first, renamed);
const spreadIt = [...[h.__getv()]];
print(spreadIt);
print(new Holder().__getv());
new Holder().__setv(1);
print(h["field"]);
