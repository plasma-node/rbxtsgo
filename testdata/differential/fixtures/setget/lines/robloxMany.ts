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

print(c.v);

const nothing = null;

// Blank lines between each.


if (c.v == 1) {
	print(nothing);
}


var old = 1;
print(old);


for (const key in { a: 1 }) {
	print(key);
}


outer: while (true) {
	break outer;
}
