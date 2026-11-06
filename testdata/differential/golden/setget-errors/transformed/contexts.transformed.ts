// Every kind of place a rewritten access can land, for the text of the reprint. Whether this type checks doesn't matter.
class Thing {
    public __getv(): number {
        return 1;
    }
    public __setv(val: number) {
        print(val);
    }
    public __getme(): Thing {
        return this;
    }
    public __setme(val: Thing) {
        print(val);
    }
    public __getfn() {
        return () => this;
    }
    public __getcls() {
        return Thing;
    }
    public __gettag() {
        return (s: TemplateStringsArray) => s;
    }
    public __getopt(): Thing | undefined {
        return this;
    }
    public field = 1;
}
const o = new Thing();
let k = 1;
async function a() {
    await o.__getv();
    await (o.__getme()).__getv();
}
function* g() {
    yield o.__getv();
    yield* o.__getme();
    o.__setv(yield);
    o.__setv(o.__getv() + (yield));
    o.__setv(o.__getv() * (yield 1));
}
print((o.__getme()).__getv());
print((((o.__getme()).__getme()).__getme()).__getv());
print((o.__getme()).field);
print((o.__getme())["field"]);
print(o.__getopt()?.__getv?.());
print(o.__getopt()?.__getme?.().__getv());
print(o.__getopt()?.__getme?.()?.__getv?.());
print(o.__getopt()?.__getme?.().__getme().field);
print((o.__getopt())!.__getv());
print(((o.__getopt())!.__getme())!.__getv());
print((o.__getme())!);
print((o.__getme())!!);
print((o.__getme()).__getv());
print((o.__getopt())?.__getv?.());
print((o.__getfn())());
print((o.__getfn())().__getv());
print((o.__getfn())()());
print(o.__getfn()?.());
print(o.__getopt()?.__getfn?.()());
print(new (o.__getcls())());
print(new (o.__getcls()));
print(new ((o.__getme()).__getcls())());
print(new (o.__getcls())());
print(new (o.__getcls())().__getv());
print((o.__gettag()) `t`);
print(((o.__getme()).__gettag()) `t`);
print(o.__getopt()?.__gettag?.());
print(o.__getv() as number);
print(o.__getme() as Thing);
print((o.__getme() as Thing).__getv());
print(<number>o.__getv());
print((<Thing>o.__getme()).__getv());
print(o.__getv() satisfies number);
print(-o.__getv());
print(!o.__getv());
print(+o.__getv(), ~o.__getv());
print(typeof o.__getv());
print(void o.__getv());
print(o.__getv() + 1, 1 + o.__getv(), o.__getv() * 2, 2 ** o.__getv(), o.__getv() ** 2);
print(o.__getv() > 1 ? o.__getv() : o.__getv());
print(o.__getv() ?? 1, o.__getv() || 1, o.__getv() && 1);
print([o.__getv()], [...[o.__getv()]], { a: o.__getv(), [o.__getv()]: 1 }, { ...o.__getme() });
print(`${o.__getv()}`, `${(o.__getme()).__getv()}x`);
print((() => o.__getv())(), (function () { return o.__getv(); })());
print(o.__getv(), k);
for (const x of [o.__getv()])
    print(x);
for (let i = o.__getv(); i < o.__getv(); o.__setv(o.__getv() + o.__getv()))
    print(i);
if (o.__getv())
    print(o.__getv());
else
    print(o.__getme());
class Derived extends (o.__getcls()) {
}
class Derived2 extends (o.__getcls()) {
}
print(Derived, Derived2);
function ret() { return o.__getv(); }
function thr() { throw o.__getv(); }
print(ret, thr);
k = o.field;
k += o.field;
o.field = 1;
(o.__getme()).field = 1;
(o.__getme()).field += 1;
(o.__getme()).field++;
(o.__getme()).__setme(o);
((o.__getme()).__getme()).__setv(1);
((o.__getme()).__getme()).__setv(((o.__getme()).__getme()).__getv() + 1);
(o.__getopt())!.__setv(1);
(o.__getopt())!.__setv((o.__getopt())!.__getv() + 1);
(o.__getme()).__setv(1);
(o.__getme()).__setv((o.__getme()).__getv() + 1);
(o.__getme() as Thing).__setv(1);
(o.__getme() as Thing).__setv((o.__getme() as Thing).__getv() + 1);
(<Thing>o.__getme()).__setv(1);
o.__setv(o.__getv() + (1 + 2));
o.__setv(o.__getv() - (1 - 2));
o.__setv(o.__getv() * 1 * 2);
o.__setv(o.__getv() / (1 / 2));
o.__setv(o.__getv() ** 1 ** 2);
o.__setv(o.__getv() + (k ? 1 : 2));
o.__setv(o.__getv() - (k ? 1 : 2));
o.__setv(o.__getv() ?? k ? 1 : 2);
o.__setv(o.__getv() || (k ? 1 : 2));
o.__setv(o.__getv() && (k ? 1 : 2));
o.__setv(o.__getv() ?? (k ?? 1));
o.__setv(o.__getv() || (k || 1));
o.__setv(o.__getv() || k && 1);
o.__setv(o.__getv() && (k || 1));
o.__setv(o.__getv() + (k, 1));
o.__setv(o.__getv() + (k = 1));
o.__setv(o.__getv() + (() => 1));
o.__setv(o.__getv() + (k as number));
o.__setv(o.__getv() ** (k as number));
o.__setv(o.__getv() ** k!);
o.__setv(o.__getv() ** -k);
o.__setv(o.__getv() ** typeof k);
o.__setv(o.__getv() ** <number>k);
o.__setv(o.__getv() * k!);
o.__setv(o.__getv() * -k);
o.__setv(o.__getv() + k++);
o.__setv(o.__getv() | k & 1);
o.__setv(o.__getv() & (k ^ 1));
o.__setv(o.__getv() ^ (k | 1));
o.__setv(o.__getv() << (k >> 1));
o.__setv(o.__getv() >> (k << 1));
o.__setv(o.__getv() >>> (k >>> 1));
o.__setv(o.__getv() % (k % 1));
o.__setv(o.__getv() / (k * 1));
o.__setv(o.__getv() * (k / 1));
o.__setv(o.__getv() - (k + 1));
o.__setv(o.__getv() + (k - 1));
o.__setv(o.__getv() + ("a" + "b"));
o.__setv(o.__getv() + o.__setv(o.__getv() + 1));
o.__setv(o.__setv(1));
o.__setv(o.__setv(1));
o.__setv(o.__getv() + o.__setv(1));
const t = o.__setv(1), u = o.__setv(o.__getv() + 1);
print(t, u);
o.__setv(o.__getv() + 1), o.__setv(o.__getv() - 1);
o.me.__setv(o.me.__getv() + 1);
(o).__setv((o).__getv() + 1);
o.__setv(o.__getv() + 1);
o.__setv(o.__getv() + 1);
o.__setv(o.__getv() + 1);
print(o.__setv(o.__getv() + 1));
print(o.__setv(o.__getv() - 1));
o["v"] = 1;
o["v"]++;
o.field++;
delete o.field;
delete (o.__getme()).field;
o.__setv([1]);
(o.__setv({ a: 1 }));
for (o.__getv() of [1])
    print(1);
for (o.__setv(0); o.__getv() < 1; o.__setv(o.__getv() + 1))
    print(1);
