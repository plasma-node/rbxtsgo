class C {
	public get x() {
		return 1;
	}
}

const { x } = new C(); //! TS2339

print(x);
