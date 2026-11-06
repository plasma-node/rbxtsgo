abstract class Shape {
	protected size = 1;

	public abstract get area(): number;

	public get doubled(): number {
		return this.size * 2;
	}

	public set doubled(val: number) {
		this.size = val / 2;
	}

	public get label(): string {
		return "shape";
	}
}

class Square extends Shape {
	public get area(): number {
		return this.size * this.size;
	}

	public override get label(): string {
		return "square " + super.label;
	}

	public grow() {
		super.doubled = 10;
		this.doubled += 1;
		return super.doubled;
	}
}

class Tiny extends Square {
	public override get doubled(): number {
		return super.doubled + 1;
	}

	public override set doubled(val: number) {
		super.doubled = val - 1;
	}
}

class GetInBaseSetHere extends Shape {
	public get area(): number {
		return 0;
	}

	public set label(val: string) {
		print(val);
	}
}

const square = new Square();
print(square.area, square.label, square.grow());
const tiny = new Tiny();
tiny.doubled = 4;
print(tiny.doubled, tiny.area);
const asShape: Shape = tiny;
print(asShape.area, asShape.label);
asShape.doubled = 8;
const other = new GetInBaseSetHere();
other.label = "x";

interface HasName {
	get name(): string;
}

class Named implements HasName {
	private inner = "n";
	public get name() {
		return this.inner;
	}
	public set name(value: string) {
		this.inner = value;
	}
}

const named: HasName = new Named();
print(named.name);
const concrete = new Named();
concrete.name = "m";

class Box<T> {
	public constructor(private item: T) {}
	public get value(): T {
		return this.item;
	}
	public set value(val: T) {
		this.item = val;
	}
}

const box = new Box<number>(1);
box.value += 2;
print(new Box("s").value, box.value);
