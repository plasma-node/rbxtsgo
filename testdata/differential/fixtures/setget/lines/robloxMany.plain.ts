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

print(c.__getv());

const nothing = null;

// Blank lines between each.


if (c.__getv() == 1) {
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
