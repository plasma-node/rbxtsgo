class Base {
	public v = 1;
}

// TS itself rejects this before the reprint, and JS accepts it, because the getter is a method by then.
class Child extends Base {
	public get v() {
		return 2;
	}
}

print(new Child().v); // JS reports nothing here: by then, the getter is a method
