class D {
	// A comment.

	public set onlySet(val: number) {
		print(val);
	}

	public get onlyGet() {
		return 1;
	}
}

const d = new D();

print(d.onlySet);

print(d.onlyGet + "x" + d.nothing);
