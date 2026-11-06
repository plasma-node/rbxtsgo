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

interface Wants {
	// The related location is down here, after the printer's shifts.

	x: number;
}

const value: Wants = new C();

print(value);
