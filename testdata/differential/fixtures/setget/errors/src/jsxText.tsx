import Roact from "@rbxts/roact";

class C {
	public get w() {
		return 1;
	}
}

const element = (
	<frame Size={new UDim2(new C().w, 0, 0, 0)}> {/*! TS2322 TS2747 */}
		text
	</frame>
);
print(element);
