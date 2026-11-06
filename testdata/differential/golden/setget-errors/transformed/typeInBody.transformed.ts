// Blank lines and comments shift what JS reports.
class C {
    public __getvalue(): string {
        return 1; //! TS2322
    }
    public __setvalue(val: string) {
        const bad: number = val; //! TS2322
        print(bad);
    }
}
print(new C().__getvalue());
