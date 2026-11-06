// Evaluation order, prereqs and temp names.
function side(n: number) {
    return n;
}
const arr = [1, 2, 3];
const map = new Map<string, number>();
const obj = { a: 1, b: { c: 2 } };
let i = 0;
export const r1 = arr[i++] + arr[i++];
export const r2 = [side(1), ...arr, side(2), i++, ...arr];
export const r3 = { x: i++, [`k${i++}`]: side(i), ...obj, y: side(3) };
export const r4 = side(i++) ? (i > 2 ? side(i++) : side(i--)) : side(0);
arr[side(0)] = arr[side(1)] += side(2);
obj.b.c = obj.a = i = 5;
export const r5 = (i = 6) + (i += 1);
map.set("a", side(1));
export const r6 = map.get("a") ?? side(2);
export const r7 = obj.a && obj.b && side(1);
export const r8 = i++ || i-- || side(0);
let maybe: {
    f?: (x: number) => number;
    g: {
        h?: number;
    };
} | undefined;
export const r9 = maybe?.f?.(side(1));
export const r10 = maybe?.g.h ?? 2;
const [p, , q = side(1)] = arr;
const { a: renamed, b: { c: deep = 3 }, } = obj;
export { p, q, renamed, deep };
for (let k = 0, m = side(1); k < m; k++) {
    const fn = () => k;
    if (fn() > 0)
        continue;
    if (k > 3)
        break;
}
for (const [key, value] of map) {
    print(key, value);
}
for (const v of arr) {
    print(v);
}
for (const c of "héllo") {
    print(c);
}
function* gen(): Generator<number, number, undefined> {
    yield 1;
    const got = yield 2;
    print(got);
    return 3;
}
for (const g of gen()) {
    print(g);
}
switch (side(1)) {
    case side(1):
    case 2: {
        print("a");
    }
    // falls through
    case 3:
        print("b");
        break;
    default:
        print("c");
}
try {
    print(side(1));
    throw "x";
}
catch (e) {
    print(e);
}
finally {
    print("done");
}
function usesTry() {
    try {
        return side(1);
    }
    catch {
        return side(2);
    }
}
export { usesTry };
print(hoisted());
function hoisted() {
    return later;
}
const later = 1;
const label = `${side(1)}-${"x"}-${i}`;
export { label };
export const typeChecks = typeOf(label) === "string" && typeIs(label, "string");
