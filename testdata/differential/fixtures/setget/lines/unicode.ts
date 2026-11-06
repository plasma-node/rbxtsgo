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

/* 😀é */ const wrong: string = c.v;
const text = "日本語😀"; const wrong2: number = text;
const 日本 = 1; const wrong3: string = 日本; print(wrong, wrong2, wrong3);
