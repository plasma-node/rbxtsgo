// JS 3.0.0 crashes on a parenthesized target in an assignment pattern, in each of the functions named below.
let a = 0;
let b = 0;
let c = 0;
let d = 0;
let e = 0;
const o = { p: 0 };
const list = [0];

// transformOptimizedArrayAssignmentPattern
[(a)] = [1];
print("simple", a);
[(a), (b)] = [2, 3];
print("two", a, b);
[(o.p)] = [4];
print("property", o.p);
[(list[0])] = [5];
print("element", list[0]);
[(c) = 6] = [];
print("default", c);
[((d))] = [7];
print("double", d);
[(d as number)] = [8];
print("as", d);
[(d!)] = [9];
print("non-null", d);

// transformArrayAssignmentPattern
[[(e)]] = [[10]];
print("nested", e);
[[(a), (b)]] = [[11, 12]];
print("nested two", a, b);

// transformInLineArrayAssignmentPattern
for ([(a), (b)] of [[13, 14]]) {
	print("for-of", a, b);
}
for ([(a)] of [[15]]) {
	print("for-of one", a);
}

// transformObjectAssignmentPattern
({ a: (a) } = { a: 16 });
print("object", a);
({ a: (o.p) } = { a: 17 });
print("object property", o.p);
({ a: (a) = 18 } = { a: undefined as number | undefined });
print("object default", a);
({ a: a! } = { a: 19 });
print("object non-null", a);
for ({ a: (a) } of [{ a: 20 }]) {
	print("object for-of", a);
}

// JS 3.0.0 also asserts that the property's name is an identifier.
({ "a": [a] } = { a: [21] });
print("quoted name", a);

export {};
