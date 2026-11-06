class C {
    public __getx() {
        return 1;
    }
}
const asObject: {
    x: number;
} = new C(); //! TS2741
print(asObject);
