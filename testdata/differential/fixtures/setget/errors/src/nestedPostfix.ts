class Inner {
	public get n() {
		return 1;
	}
	public set n(val: number) {
		print(val);
	}
}
class Outer {
	public get inner() {
		return new Inner();
	}
}
const o = new Outer();
// A postfix operator keeps the object expression as written, so the getter isn't rewritten inside it.
o.inner.n++; //! TS2339
