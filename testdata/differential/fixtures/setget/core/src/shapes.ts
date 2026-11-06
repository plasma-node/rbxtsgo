class Holder {
	private inner = 1;
	public get v(): number {
		return this.inner;
	}
	public set v(val: number) {
		this.inner = val;
	}
	public get val(): Holder | undefined {
		return this;
	}
	public set val(value: Holder | undefined) {
		print(value);
	}
	public field = 1;
	public maybe?: Holder;
}
const h = new Holder();
const maybe = h as Holder | undefined;

// optional chaining
print(maybe?.v);
print(maybe?.val?.v);
print(h.val?.v);
print(h.maybe?.val?.field);
print(h.maybe?.field);

// non-null, parens and casts
print(h.val!.v);
h.val!.v = 2;
h.val!.v += 2;
(h).v = 3;
(h.v) = 4;
(h as Holder).v = 5;
(h.v as number) = 6;
print((h as Holder).v);
print((h).v);

// not a set when it's only under the assignment's right side
const arr = [1, 2];
arr[h.v] = 1;
let plain = 0;
plain = h.field;
print(plain);
const obj = { key: 1 };
obj.key = h.field;
obj.key += h.field;
print(obj);

// the first property access anywhere under the expression decides
plain = h.v;
plain += h.v;
print(plain);
arr[0] = h.v;
const y = (plain = h.v);
print(y);
let z = h.v > 1 ? h.v : 0;
z = h.v > 1 ? h.field : 0;
print(z);
print(h.v + h.v, h.v === 1, h.v !== h.field);
h.v = h.v + 1;
h.v = h.field;
h.val = h;
h.val = h.val;
h.field = h.v;
const nested = (h.v = 1);
print(nested);

// calls and declarations
function take(a: number, b?: number) {
	return a + (b ?? 0);
}
print(take(h.v), take(h.v, h.v));
const fn = () => h.v;
print(fn());
const typed: number = h.v;
print(typed);
const [first] = [h.v];
const { key: renamed } = { key: h.v };
print(first, renamed);
const spreadIt = [...[h.v]];
print(spreadIt);
print(new Holder().v);
new Holder().v = 1;
print(h["field"]);
