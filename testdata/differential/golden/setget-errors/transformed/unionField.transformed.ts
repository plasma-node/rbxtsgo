class WithGetter {
    public __getv() {
        return 1;
    }
}
class WithField {
    public v = 2;
}
const either = (math.random() > 0.5 ? new WithGetter() : new WithField()) as WithGetter | WithField;
print(either.__getv()); //! TS2339
