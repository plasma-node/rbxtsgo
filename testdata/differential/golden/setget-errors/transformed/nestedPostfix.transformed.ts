class Inner {
    public __getn() {
        return 1;
    }
    public __setn(val: number) {
        print(val);
    }
}
class Outer {
    public __getinner() {
        return new Inner();
    }
}
const o = new Outer();
// A postfix operator keeps the object expression as written, so the getter isn't rewritten inside it.
o.inner.__setn(o.inner.__getn() + 1); //! TS2339
