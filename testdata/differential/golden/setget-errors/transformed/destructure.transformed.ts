class C {
    public __getx() {
        return 1;
    }
}
const { x } = new C(); //! TS2339
print(x);
