class Left {
    public __getshared(): number {
        return 1;
    }
    public __setshared(val: number) {
        print(val);
    }
}
class Right {
    public __getshared(): number {
        return 2;
    }
    public __setshared(val: number) {
        print(val);
    }
}
const either = (math.random() > 0.5 ? new Left() : new Right()) as Left | Right;
print(either.__getshared());
either.__setshared(1);
either.__setshared(either.__getshared() + 1);
type Both = Left & {
    more: number;
};
declare const both: Both;
print(both.__getshared(), both.more);
function generic<T extends Left>(value: T) {
    value.__setshared(1);
    return value.__getshared();
}
print(generic(new Left()));
