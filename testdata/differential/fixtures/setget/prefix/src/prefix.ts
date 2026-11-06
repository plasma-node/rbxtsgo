class P {
	private inner = 1;
	public get a(): number {
		return this.inner;
	}
	public set a(val: number) {
		this.inner = val;
	}
}
const p = new P();
p.a = 1;
p.a += 1;
p.a++;
print(p.a);
