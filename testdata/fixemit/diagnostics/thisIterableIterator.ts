// The `this` type of #2371. The original extends Map, which is now noMacroExtends before it gets this far.
interface Values extends IterableIterator<number> {}
function f(this: Values) {
	for (const value of this) {
		print(value);
	}
}
export = f;
