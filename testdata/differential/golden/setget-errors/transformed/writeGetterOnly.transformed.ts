class C {
    public __getonlyGet() {
        return 1;
    }
}
const c = new C();
c.__getonlyGet() = 2; //! TS2364
