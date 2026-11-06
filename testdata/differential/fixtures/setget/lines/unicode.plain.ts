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

/* 😀é */ const wrong: string = c.__getv();
const text = "日本語😀"; const wrong2: number = text;
const 日本 = 1; const wrong3: string = 日本; print(wrong, wrong2, wrong3);
