class C {
	public get n() {
		return 1;
	}
	public set n(val: number) {
		print(val);
	}
}
const c = new C();

// A postfix operator becomes a call to the setter, which returns nothing.
print(c.n++ + 1); //! TS2365
