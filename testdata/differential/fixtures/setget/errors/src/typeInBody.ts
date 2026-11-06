// Blank lines and comments shift what JS reports.


class C {

	public get value(): string {

		return 1; //! TS2322
	}



	public set value(val: string) {
		const bad: number = val; //! TS2322
		print(bad);
	}
}



print(new C().value);
