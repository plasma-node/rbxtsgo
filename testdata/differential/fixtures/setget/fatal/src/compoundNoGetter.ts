class C {
	public set onlySet(val: number) {
		print(val);
	}
}
const c = new C();
c.onlySet += 1;
