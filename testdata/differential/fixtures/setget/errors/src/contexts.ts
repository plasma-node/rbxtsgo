// Every kind of place a rewritten access can land, for the text of the reprint. Whether this type checks doesn't matter.
class Thing {
	public get v(): number {
		return 1;
	}
	public set v(val: number) {
		print(val);
	}
	public get me(): Thing {
		return this;
	}
	public set me(val: Thing) {
		print(val);
	}
	public get fn() {
		return () => this;
	}
	public get cls() {
		return Thing;
	}
	public get tag() {
		return (s: TemplateStringsArray) => s;
	}
	public get opt(): Thing | undefined {
		return this;
	}
	public field = 1;
}
const o = new Thing();
let k = 1;
async function a() {
	await o.v;
	await o.me.v;
}
function* g() {
	yield o.v;
	yield* o.me;
	o.v = yield;
	o.v += yield;
	o.v *= yield 1;
}
print(o.me.v);
print(o.me.me.me.v);
print(o.me.field);
print(o.me["field"]);
print(o.opt?.v);
print(o.opt?.me.v);
print(o.opt?.me?.v);
print(o.opt?.me.me.field);
print(o.opt!.v);
print(o.opt!.me!.v);
print(o.me!);
print(o.me!!);
print((o.me).v);
print((o.opt)?.v);
print(o.fn());
print(o.fn().v);
print(o.fn()());
print(o.fn?.());
print(o.opt?.fn());
print(new o.cls());
print(new o.cls);
print(new o.me.cls());
print(new (o.cls)());
print(new o.cls().v);
print(o.tag`t`);
print(o.me.tag`t`);
print(o.opt?.tag);
print(o.v as number);
print(o.me as Thing);
print((o.me as Thing).v);
print(<number>o.v);
print((<Thing>o.me).v);
print(o.v satisfies number);
print(-o.v);
print(!o.v);
print(+o.v, ~o.v);
print(typeof o.v);
print(void o.v);
print(o.v + 1, 1 + o.v, o.v * 2, 2 ** o.v, o.v ** 2);
print(o.v > 1 ? o.v : o.v);
print(o.v ?? 1, o.v || 1, o.v && 1);
print([o.v], [...[o.v]], { a: o.v, [o.v]: 1 }, { ...o.me });
print(`${o.v}`, `${o.me.v}x`);
print((() => o.v)(), (function () { return o.v; })());
print(o.v, k);
for (const x of [o.v]) print(x);
for (let i = o.v; i < o.v; i += o.v) print(i);
if (o.v) print(o.v); else print(o.me);
class Derived extends o.cls {}
class Derived2 extends (o.cls) {}
print(Derived, Derived2);
function ret() { return o.v; }
function thr() { throw o.v; }
print(ret, thr);
k = o.field;
k += o.field;
o.field = 1;
o.me.field = 1;
o.me.field += 1;
o.me.field++;
o.me.me = o;
o.me.me.v = 1;
o.me.me.v += 1;
o.opt!.v = 1;
o.opt!.v += 1;
(o.me).v = 1;
(o.me).v += 1;
(o.me as Thing).v = 1;
(o.me as Thing).v += 1;
(<Thing>o.me).v = 1;
o.v += 1 + 2;
o.v -= 1 - 2;
o.v *= 1 * 2;
o.v /= 1 / 2;
o.v **= 1 ** 2;
o.v += k ? 1 : 2;
o.v -= k ? 1 : 2;
o.v ??= k ? 1 : 2;
o.v ||= k ? 1 : 2;
o.v &&= k ? 1 : 2;
o.v ??= k ?? 1;
o.v ||= k || 1;
o.v ||= k && 1;
o.v &&= k || 1;
o.v += (k, 1);
o.v += k = 1;
o.v += () => 1;
o.v += k as number;
o.v **= k as number;
o.v **= k!;
o.v **= -k;
o.v **= typeof k;
o.v **= <number>k;
o.v *= k!;
o.v *= -k;
o.v += k++;
o.v |= k & 1;
o.v &= k ^ 1;
o.v ^= k | 1;
o.v <<= k >> 1;
o.v >>= k << 1;
o.v >>>= k >>> 1;
o.v %= k % 1;
o.v /= k * 1;
o.v *= k / 1;
o.v -= k + 1;
o.v += k - 1;
o.v += "a" + "b";
o.v += o.v += 1;
o.v = o.v = 1;
k = o.v = 1;
k += o.v = 1;
const t = o.v = 1, u = o.v += 1;
print(t, u);
o.v++, o.v--;
o.me.v++;
(o).v++;
(o.v)++;
(o.v as number)++;
o.v!++;
print(o.v++);
print(o.v--);
o["v"] = 1;
o["v"]++;
o.field++;
delete o.field;
delete o.me.field;
[o.v] = [1];
({ a: o.v } = { a: 1 });
for (o.v of [1]) print(1);
for (o.v = 0; o.v < 1; o.v++) print(1);
