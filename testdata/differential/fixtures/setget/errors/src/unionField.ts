class WithGetter {
	public get v() {
		return 1;
	}
}

class WithField {
	public v = 2;
}

const either = (math.random() > 0.5 ? new WithGetter() : new WithField()) as WithGetter | WithField;

print(either.v); //! TS2339
