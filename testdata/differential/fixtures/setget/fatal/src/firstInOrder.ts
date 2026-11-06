class C {
	public set a(val: number) {
		print(val);
	}
	public set b(val: number) {
		print(val);
	}
}
const c = new C();
c.a += 1;
c.b++;
