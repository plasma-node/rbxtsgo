class Left {
	public get shared(): number {
		return 1;
	}
	public set shared(val: number) {
		print(val);
	}
}
class Right {
	public get shared(): number {
		return 2;
	}
	public set shared(val: number) {
		print(val);
	}
}
const either = (math.random() > 0.5 ? new Left() : new Right()) as Left | Right;
print(either.shared);
either.shared = 1;
either.shared += 1;

type Both = Left & { more: number };
declare const both: Both;
print(both.shared, both.more);

function generic<T extends Left>(value: T) {
	value.shared = 1;
	return value.shared;
}
print(generic(new Left()));
