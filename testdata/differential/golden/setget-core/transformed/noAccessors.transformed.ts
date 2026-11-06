/*
 * This file has no accessors. JS reprints it anyway.
 *        These lines get new indents.
 */
// numeric spelling
print(1000, 0x1F, 0X1f, 0b101, 0B101, 0o17, 1e3, 1E3, 1.50, .5, 5., 10.01);
const big = 65535;
// string spelling
print('single', "double", `template`, 'it\'s', "say \"hi\"", '\x41B\u{43}', "é日本語😀");
print(`a${big}b`, `multi
line`);
print("\
continued");
// blank lines and semicolons
const a = 1;
const b = 2;
;
;
print(a, b);
// comments in odd places
const o = {
    // first
    x: 1, // after x
    /* before y */ y: 2,
    // last
};
const arr = [
    // in an array
    1,
    2, // two
    // end of an array
];
print(o, arr);
function f( /* no params */) {
    /* only a comment */
}
f( /* no args */);
class Empty {
}
print(new Empty());
switch (a as number) {
    // before a case
    case 1: // after a case label
        // in a case
        print(1);
        break;
    // before a default
    default:
        print(2);
    // after a default
}
for (const v of arr) /* after for */ {
    print(v); // in a loop
}
const fn = (x: number) => /* before the body */ x + 1;
print(fn(1));
const t = /* before */ a /* mid */ + /* mid2 */ b /* after */;
print(t);
// parentheses stay
print((1 + 2) * 3, (a), ((b)));
print(a! + 1, (a as number)!);
print(arr[0]!, o!.x, (o!).y);
// types go through the printer too
type Fn = (a: number) => void;
interface I<T> {
    a: T;
    b?: string;
    readonly c: number;
}
type U = {
    a: 1;
} | {
    b: 2;
};
declare const u: U;
print(typeOf(u));
export {};
// last comment
