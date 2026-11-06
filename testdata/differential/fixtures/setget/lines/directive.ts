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


// @ts-ignore
const ignored: string = 1;


// @ts-expect-error
const expected: string = 1;


// @ts-expect-error
const notAnError: string = "x";


const wrong: string = 1;
print(ignored, expected, notAnError, wrong);
