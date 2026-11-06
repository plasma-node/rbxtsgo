/**
 * A header comment.
 *        Indented.
 */

class D {

	public __getv(): string {

		const unused = 1;

		return unused;
	}



	public __setv(val: string) {

		const bad: number = val;
		print(bad);
	}
}


print(new D());
