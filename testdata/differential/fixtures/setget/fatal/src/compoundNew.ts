class C {
	public get v() {
		return 1;
	}
	public set v(val: number) {
		print(val);
	}
}
new C().v += 1;
