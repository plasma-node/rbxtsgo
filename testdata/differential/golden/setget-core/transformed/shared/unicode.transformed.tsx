import Roact from "@rbxts/roact";
class Unicode {
    public __getword(): string {
        return "日本語😀";
    }
    public __setword(val: string) {
        print(val);
    }
}
const u = new Unicode();
u.__setword("😀");
print(u.__getword(), u.__getword());
class Sized {
    public __getwidth() {
        return 1;
    }
}
const sized = new Sized();
// JSX goes through the printer as well
const element = (<frame 
// a comment in the attributes
Size={new UDim2(sized.__getwidth(), 0, 0, 0)} Key="k">
		<textlabel Text={"a" + u.__getword()}/>
		{/* a comment in the children */}
	</frame>);
print(element);
