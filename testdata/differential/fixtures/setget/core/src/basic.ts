export class Basic {
	private value = 1;
	private other = "text";

	public get both(): number {
		return this.value;
	}

	public set both(val: number) {
		this.value = val;
	}

	public get onlyGet(): string {
		return this.other;
	}

	public set onlySet(val: string) {
		this.other = val;
	}

	public static count = 0;

	public static get total(): number {
		return Basic.count;
	}

	public static set total(val: number) {
		Basic.count = val;
	}

	public viaThis() {
		this.both = this.both + 1;
		this.onlySet = this.onlyGet;
		return this.both;
	}
}

const basic = new Basic();
basic.both = 5;
print(basic.both);
print(basic.onlyGet);
basic.onlySet = "val";
Basic.total = 3;
print(Basic.total);
print(basic.viaThis());
print(new Basic().both);
const fromCall = () => basic;
print(fromCall().both);
fromCall().both = 2;
print(typeOf(basic.both));
print(`${basic.both} and ${basic.onlyGet}`);
const list = [basic.both, basic.both];
const record = { a: basic.both, b: basic.onlyGet };
for (let i = basic.both; i < basic.both + 1; i++) {
	print(i, list, record);
}
if (basic.both > 1) {
	print("big");
}
while (basic.both > 100) {
	basic.both = 0;
}
function usesDefault(a = basic.both) {
	return a;
}
print(usesDefault());
