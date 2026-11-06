import Roact from "@rbxts/roact";

class Unicode {
	public get word(): string {
		return "日本語😀";
	}
	public set word(val: string) {
		print(val);
	}
}
const u = new Unicode();
u.word = "😀";
print(u.word, /* 😀 */ u.word);

class Sized {
	public get width() {
		return 1;
	}
}
const sized = new Sized();
// JSX goes through the printer as well
const element = (
	<frame
		// a comment in the attributes
		Size={new UDim2(sized.width, 0, 0, 0)}
		Key="k"
	>
		<textlabel Text={"a" + u.word} />
		{/* a comment in the children */}
	</frame>
);
print(element);
