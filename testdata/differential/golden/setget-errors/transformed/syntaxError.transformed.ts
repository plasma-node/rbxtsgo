class C {
    public __getv() {
        return 1;
    }
}
const c = new C();
print(c.__getv()); // JS reports nothing: the printer repairs the missing parenthesis
