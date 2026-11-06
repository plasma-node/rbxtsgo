/**
 * A header comment.
 *        Indented.
 */

class D {

	public get v(): string {

		const unused = 1;

		return unused;
	}



	public set v(val: string) {

		const bad: number = val;
		print(bad);
	}
}


print(new D());
