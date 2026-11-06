import Roact from "@rbxts/roact";
abstract class Base {
    static count = 0;
    public field = 1;
    constructor(public readonly name: string, protected size = 2) { }
    abstract act(): void;
    greet() {
        return "hi " + this.name;
    }
    static make() {
        return Base.count++;
    }
    toString() {
        return this.name;
    }
}
export class Child extends Base {
    extra = "x";
    act() {
        print(super.greet());
    }
}
function dec(c: object) { }
function propDec(target: object, key: string) { }
@dec
export class Decorated {
    @propDec
    prop = 1;
}
const Elt = (<frame Size={new UDim2(0, 1, 0, 1)} Key="k" Event={{ MouseEnter: () => print("&amp; é") }}>
		<textlabel Text="a &amp; 😀 b"/>
		<textbutton Text={"x"}/>
		{[1, 2].map(n => (<textlabel Text={`${n}`}/>))}
	</frame>);
export { Elt };
