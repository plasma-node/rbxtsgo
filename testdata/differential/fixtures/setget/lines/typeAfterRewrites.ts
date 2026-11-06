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

const wrong: string = 1;

// More text, more blank lines.


function f(a: number) {
	return a;
}
f("x");
print(wrong, c.v);
