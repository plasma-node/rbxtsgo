class Counter {
	private inner = 1;
	public get n(): number {
		return this.inner;
	}
	public set n(val: number) {
		this.inner = val;
	}
	public get flag(): boolean | undefined {
		return true;
	}
	public set flag(val: boolean | undefined) {
		print(val);
	}
	public child?: Counter;
	public get self(): Counter {
		return this;
	}
	public set self(val: Counter) {
		print(val);
	}
}
const c = new Counter();
let k = 2;
c.n += 1;
c.n -= 1;
c.n *= 2;
c.n /= 2;
c.n %= 2;
c.n **= 2;
c.n <<= 1;
c.n >>= 1;
c.n >>>= 1;
c.n &= 1;
c.n |= 1;
c.n ^= 1;
c.flag ||= true;
c.flag &&= false;
c.flag ??= true;
c.n += k + 1;
c.n -= k - 1;
c.n -= k + 1;
c.n *= k + 1;
c.n *= k * 2;
c.n /= k / 2;
c.n **= k ** 2;
c.n **= k * 2;
c.n &= k & 1;
c.n &= k | 1;
c.n |= k | 1;
c.n ^= k ^ 1;
c.n <<= k << 1;
c.n += (k + 1);
c.n += k > 1 ? 1 : 2;
c.n += c.n;
c.n *= c.n + c.n;
c.flag ||= k > 1 && k < 3;
c.flag &&= k > 1 || k < 3;
c.flag ??= k > 1 ? true : undefined;
c.self.n += 1;
c.self.self.n *= 2;
c.n++;
c.n--;
k = c.n;
k = c.n + 1;
const before = c.n++;
print(before, k);
for (let i = 0; i < 2; c.n++) {
	i++;
}
function getCounter() {
	print("called");
	return c;
}
getCounter().n += 1;
getCounter().n++;
const arr = [c];
arr[0].n += 1;
arr[0].n++;
arr[k].n = 3;
