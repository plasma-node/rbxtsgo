/*
 * A header the printer re-indents.
 *        Like this.
 */

class C {
	private inner = 1;

	// A comment and blank lines that the printer changes.


	public __getv(): number {

		return this.inner;
	}


	public __setv(val: number) {
		this.inner = val;
	}
}


const c = new C();
c.__setv(1);

c.__setv(c.__getv() + 1);
c.__setv(c.__getv() + 1);
print(c.__getv());

function f(a: number, b: number) {
	return a + b;
}

/*
		A comment that gets new indents,
			and takes lines the same way.
*/
print(
	// a comment in the arguments
	c.__getv(),

	f(
		1,

		"x",
	),
);

const o = {
	// first

	a: 1,

	b: f(1, "y"),
};
print(o);
