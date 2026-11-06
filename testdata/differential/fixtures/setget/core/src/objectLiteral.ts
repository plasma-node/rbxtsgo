let store = 1;
const literal = {
	plain: 1,
	get value() {
		return store;
	},
	set value(val: number) {
		store = val;
	},
	get readOnly() {
		return "r";
	},
	method() {
		return this.value;
	},
};
literal.value = 2;
literal.value += 1;
literal.value++;
print(literal.value, literal.readOnly, literal.method(), literal.plain);

const single = { get one() { return 1; } };
print(single.one);
