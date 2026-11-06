class C {
    public __getn() {
        return 1;
    }
    public __setn(val: number) {
        print(val);
    }
}
const c = new C();
// A postfix operator becomes a call to the setter, which returns nothing.
print(c.__setn(c.__getn() + 1) + 1); //! TS2365
