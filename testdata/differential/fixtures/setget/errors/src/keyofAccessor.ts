class C {
	public get x() {
		return 1;
	}
}

const k: keyof C = "x"; //! TS2322

print(k);
