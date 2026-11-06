class C {
    public __getv() {
        return 1;
    }
    public __setv(val: number) {
        print(val);
    }
}
const o = new C();
const k = 1;
// This prints as a ?? b || c, which TS rejects. TS 5.5.3 and TS 7 report it at different tokens of the same line.
o.__setv(o.__getv() ?? k || 1); //! TS5076
// This one gets parentheses, so it's fine on both.
o.__setv(o.__getv() || (k ?? 1));
