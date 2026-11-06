interface HasX {
	x: number;
}

// The accessor becomes a method, so the class doesn't have x any more.
class C implements HasX { //! TS2420
	public get x() {
		return 1;
	}
}

print(new C());
