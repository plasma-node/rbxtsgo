class C {
    public __getx() {
        return 1;
    }
}
const k: keyof C = "x"; //! TS2322
print(k);
