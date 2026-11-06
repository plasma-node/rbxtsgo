class Left {
	public get shared(): number {
		return 1;
	}
}
const partial: Partial<Left> = {};
print(partial.shared); //! TS2722
