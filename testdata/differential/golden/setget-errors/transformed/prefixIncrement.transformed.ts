class C {
    public __getx() {
        return 1;
    }
    public __setx(val: number) {
        print(val);
    }
}
const c = new C();
++c.__getx(); //! TS2357
