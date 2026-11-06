class C {
    public __getv(): number {
        return 1;
    }
}
const c = new C();
print(c.__getv());
function f() {
    return null; //! roblox-ts
}
print(f());
