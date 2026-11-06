// Unions whose members have different symbols, because getFirstDefinedSymbol reads the first member.
// Each case is written in both member orders. See docs/union-order.md.
interface HasSize {
	size(): number;
}
class ClassWithSize {
	size() {
		return 1;
	}
}
declare const macroOrCustom: Array<string> | HasSize;
declare const customOrMacro: HasSize | Array<string>;
declare const macroOrClass: ReadonlyArray<string> | ClassWithSize;
declare const classOrMacro: ClassWithSize | ReadonlyArray<string>;
declare const tupleOrCustom: [number, string] | HasSize;
declare const customOrTuple: HasSize | [number, string];

export const a = macroOrCustom.size();
export const b = customOrMacro.size();
export const c = macroOrClass.size();
export const d = classOrMacro.size();
export const e = tupleOrCustom.size();
export const f = customOrTuple.size();

// A call macro in a union with a plain function.
declare const assertOrPlain: typeof assert | ((value: unknown, message?: string) => void);
declare const plainOrAssert: ((value: unknown, message?: string) => void) | typeof assert;
assertOrPlain(true);
plainOrAssert(true);

// The same interface declared before and after the array in a union, used as a value and with an optional call.
declare const optionalUnion: Array<string> | HasSize | undefined;
export const g = optionalUnion?.size();
declare const optionalReverse: undefined | HasSize | Array<string>;
export const h = optionalReverse?.size();
