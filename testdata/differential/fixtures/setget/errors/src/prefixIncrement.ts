class C {
	public get x() {
		return 1;
	}
	public set x(val: number) {
		print(val);
	}
}

const c = new C();

++c.x; //! TS2357
