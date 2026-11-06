class C {
	public get n() {
		return 1;
	}
	public set n(val: number) {
		print(val);
	}
}
const c = new C();
function* gen() {
	c.n **= yield; //! TS1109 TS2554 TS7057
	c.n += yield 1; //! TS7057
}
print(gen);
