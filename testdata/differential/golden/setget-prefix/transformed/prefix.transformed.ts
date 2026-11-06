class P {
    private inner = 1;
    public _geta(): number {
        return this.inner;
    }
    public _seta(val: number) {
        this.inner = val;
    }
}
const p = new P();
p._seta(1);
p._seta(p._geta() + 1);
p._seta(p._geta() + 1);
print(p._geta());
