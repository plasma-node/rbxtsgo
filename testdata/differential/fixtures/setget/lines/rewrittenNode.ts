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

function takesString(s: string) {
	return s;
}
const s: string = c.v;
takesString(c.v);
takesString("a", c.v);
c.v = "text";
print(s);
