import Roact from "@rbxts/roact";
class C {
    public __getw() {
        return 1;
    }
}
const element = (<frame Size={new UDim2(new C().__getw(), 0, 0, 0)}> {/*! TS2322 TS2747 */}
		text
	</frame>);
print(element);
