class C {
    public __getn() {
        return 1;
    }
    public __setn(val: number) {
        print(val);
    }
}
const c = new C();
function* gen() {
    c.__setn(c.__getn() ** yield); //! TS1109 TS2554 TS7057
    c.__setn(c.__getn() + (yield 1)); //! TS7057
}
print(gen);
