class C {
    public __setonlySet(val: number) {
        print(val);
    }
}
const c = new C();
print(c.__getonlySet()); //! TS2551
