class D {
	// A comment.

	public __setonlySet(val: number) {
		print(val);
	}

	public __getonlyGet() {
		return 1;
	}
}

const d = new D();

print(d.__getonlySet());

print(d.__getonlyGet() + "x" + d.nothing);
