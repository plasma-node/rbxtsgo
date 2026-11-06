class C {
	public get x() {
		return 1;
	}
}

// The property is called yx here, and it keeps the declaration of the getter x.
declare const renamed: { [K in keyof C as `y${K}`]: C[K] };

print(renamed.yx); //! TS2339
