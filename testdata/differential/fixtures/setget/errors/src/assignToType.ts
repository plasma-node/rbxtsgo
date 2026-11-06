class C {
	public get x() {
		return 1;
	}
}

const asObject: { x: number } = new C(); //! TS2741

print(asObject);
