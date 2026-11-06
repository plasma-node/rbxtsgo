let store = 1;
const literal = {
    plain: 1,
    __getvalue() {
        return store;
    },
    __setvalue(val: number) {
        store = val;
    },
    __getreadOnly() {
        return "r";
    },
    method() {
        return this.__getvalue();
    },
};
literal.__setvalue(2);
literal.__setvalue(literal.__getvalue() + 1);
literal.__setvalue(literal.__getvalue() + 1);
print(literal.__getvalue(), literal.__getreadOnly(), literal.method(), literal.plain);
const single = { __getone() { return 1; } };
print(single.__getone());
