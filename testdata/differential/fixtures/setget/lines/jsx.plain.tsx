import Roact from "@rbxts/roact";

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

const element = (
	<frame
		// a comment

		Size={c.__getv()}
	>
		<textlabel

			Text={1}
		/>
	</frame>
);
print(element);
