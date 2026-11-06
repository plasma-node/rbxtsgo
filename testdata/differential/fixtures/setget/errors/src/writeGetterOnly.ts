class C {
	public get onlyGet() {
		return 1;
	}
}

const c = new C();

c.onlyGet = 2; //! TS2364
