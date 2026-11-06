/*
 * A header the printer re-indents.
 *        Like this.
 */

class C {
	private inner = 1;

	// A comment and blank lines that the printer changes.


	public get v(): number {

		return this.inner;
	}


	public set v(val: number) {
		this.inner = val;
	}
}


const c = new C();
c.v = 1;

c.v += 1;
c.v++;
print(c.v);

function f(a: number, b: number) {
	return a + b;
}

/*
		A comment that gets new indents,
			and takes lines the same way.
*/
print(
	// a comment in the arguments
	c.v,

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
