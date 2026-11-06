class Counter {
    private inner = 1;
    public __getn(): number {
        return this.inner;
    }
    public __setn(val: number) {
        this.inner = val;
    }
    public __getflag(): boolean | undefined {
        return true;
    }
    public __setflag(val: boolean | undefined) {
        print(val);
    }
    public child?: Counter;
    public __getself(): Counter {
        return this;
    }
    public __setself(val: Counter) {
        print(val);
    }
}
const c = new Counter();
let k = 2;
c.__setn(c.__getn() + 1);
c.__setn(c.__getn() - 1);
c.__setn(c.__getn() * 2);
c.__setn(c.__getn() / 2);
c.__setn(c.__getn() % 2);
c.__setn(c.__getn() ** 2);
c.__setn(c.__getn() << 1);
c.__setn(c.__getn() >> 1);
c.__setn(c.__getn() >>> 1);
c.__setn(c.__getn() & 1);
c.__setn(c.__getn() | 1);
c.__setn(c.__getn() ^ 1);
c.__setflag(c.__getflag() || true);
c.__setflag(c.__getflag() && false);
c.__setflag(c.__getflag() ?? true);
c.__setn(c.__getn() + (k + 1));
c.__setn(c.__getn() - (k - 1));
c.__setn(c.__getn() - (k + 1));
c.__setn(c.__getn() * (k + 1));
c.__setn(c.__getn() * k * 2);
c.__setn(c.__getn() / (k / 2));
c.__setn(c.__getn() ** k ** 2);
c.__setn(c.__getn() ** (k * 2));
c.__setn(c.__getn() & k & 1);
c.__setn(c.__getn() & (k | 1));
c.__setn(c.__getn() | k | 1);
c.__setn(c.__getn() ^ k ^ 1);
c.__setn(c.__getn() << (k << 1));
c.__setn(c.__getn() + (k + 1));
c.__setn(c.__getn() + (k > 1 ? 1 : 2));
c.__setn(c.__getn() + c.__getn());
c.__setn(c.__getn() * (c.__getn() + c.__getn()));
c.__setflag(c.__getflag() || k > 1 && k < 3);
c.__setflag(c.__getflag() && (k > 1 || k < 3));
c.__setflag(c.__getflag() ?? k > 1 ? true : undefined);
(c.__getself()).__setn((c.__getself()).__getn() + 1);
((c.__getself()).__getself()).__setn(((c.__getself()).__getself()).__getn() * 2);
c.__setn(c.__getn() + 1);
c.__setn(c.__getn() - 1);
c.__setn(c.__getn());
c.__setn(c.__getn() + 1);
const before = c.__setn(c.__getn() + 1);
print(before, k);
for (let i = 0; i < 2; c.__setn(c.__getn() + 1)) {
    i++;
}
function getCounter() {
    print("called");
    return c;
}
getCounter().__setn(getCounter().__getn() + 1);
getCounter().__setn(getCounter().__getn() + 1);
const arr = [c];
arr[0].__setn(arr[0].__getn() + 1);
arr[0].__setn(arr[0].__getn() + 1);
arr[k].__setn(3);
