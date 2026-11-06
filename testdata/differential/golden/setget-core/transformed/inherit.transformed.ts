abstract class Shape {
    protected size = 1;
    public abstract __getarea(): number;
    public __getdoubled(): number {
        return this.size * 2;
    }
    public __setdoubled(val: number) {
        this.size = val / 2;
    }
    public __getlabel(): string {
        return "shape";
    }
}
class Square extends Shape {
    public __getarea(): number {
        return this.size * this.size;
    }
    public override __getlabel(): string {
        return "square " + super.__getlabel();
    }
    public grow() {
        super.__setdoubled(10);
        this.__setdoubled(this.__getdoubled() + 1);
        return super.__getdoubled();
    }
}
class Tiny extends Square {
    public override __getdoubled(): number {
        return super.__getdoubled() + 1;
    }
    public override __setdoubled(val: number) {
        super.__setdoubled(val - 1);
    }
}
class GetInBaseSetHere extends Shape {
    public __getarea(): number {
        return 0;
    }
    public __setlabel(val: string) {
        print(val);
    }
}
const square = new Square();
print(square.__getarea(), square.__getlabel(), square.grow());
const tiny = new Tiny();
tiny.__setdoubled(4);
print(tiny.__getdoubled(), tiny.__getarea());
const asShape: Shape = tiny;
print(asShape.__getarea(), asShape.__getlabel());
asShape.__setdoubled(8);
const other = new GetInBaseSetHere();
other.__setlabel("x");
interface HasName {
    __getname(): string;
}
class Named implements HasName {
    private inner = "n";
    public __getname() {
        return this.inner;
    }
    public __setname(value: string) {
        this.inner = value;
    }
}
const named: HasName = new Named();
print(named.__getname());
const concrete = new Named();
concrete.__setname("m");
class Box<T> {
    public constructor(private item: T) { }
    public __getvalue(): T {
        return this.item;
    }
    public __setvalue(val: T) {
        this.item = val;
    }
}
const box = new Box<number>(1);
box.__setvalue(box.__getvalue() + 2);
print(new Box("s").__getvalue(), box.__getvalue());
