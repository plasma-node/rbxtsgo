class C {
	public get x() {
		return 1;
	}
}

const c = new C();

print(c["x"]); //! TS7053
