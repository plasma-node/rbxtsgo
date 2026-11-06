// Once any file augments a global, that global's symbol is cloned per checker. See 0002.
declare global {
	interface Array<T> {
		augmented(this: Array<T>): void;
	}
}

export const values: Array<number> = [];
export const size = values.size();
