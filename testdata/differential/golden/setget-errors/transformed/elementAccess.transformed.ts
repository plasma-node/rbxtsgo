class C {
    public __getx() {
        return 1;
    }
}
const c = new C();
print(c["x"]); //! TS7053
