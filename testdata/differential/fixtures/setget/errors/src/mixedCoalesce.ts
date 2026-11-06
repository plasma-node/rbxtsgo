class C {
	public get v() {
		return 1;
	}
	public set v(val: number) {
		print(val);
	}
}
const o = new C();
const k = 1;

// This prints as a ?? b || c, which TS rejects. TS 5.5.3 and TS 7 report it at different tokens of the same line.
o.v ??= k || 1; //! TS5076
// This one gets parentheses, so it's fine on both.
o.v ||= k ?? 1;
