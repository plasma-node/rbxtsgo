class C {
	public set onlySet(val: number) {
		print(val);
	}
}

const c = new C();

print(c.onlySet); //! TS2551
