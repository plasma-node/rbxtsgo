export class Basic {
    private value = 1;
    private other = "text";
    public __getboth(): number {
        return this.value;
    }
    public __setboth(val: number) {
        this.value = val;
    }
    public __getonlyGet(): string {
        return this.other;
    }
    public __setonlySet(val: string) {
        this.other = val;
    }
    public static count = 0;
    public static __gettotal(): number {
        return Basic.count;
    }
    public static __settotal(val: number) {
        Basic.count = val;
    }
    public viaThis() {
        this.__setboth(this.__getboth() + 1);
        this.__setonlySet(this.__getonlyGet());
        return this.__getboth();
    }
}
const basic = new Basic();
basic.__setboth(5);
print(basic.__getboth());
print(basic.__getonlyGet());
basic.__setonlySet("val");
Basic.__settotal(3);
print(Basic.__gettotal());
print(basic.viaThis());
print(new Basic().__getboth());
const fromCall = () => basic;
print(fromCall().__getboth());
fromCall().__setboth(2);
print(typeOf(basic.__getboth()));
print(`${basic.__getboth()} and ${basic.__getonlyGet()}`);
const list = [basic.__getboth(), basic.__getboth()];
const record = { a: basic.__getboth(), b: basic.__getonlyGet() };
for (let i = basic.__getboth(); i < basic.__getboth() + 1; i++) {
    print(i, list, record);
}
if (basic.__getboth() > 1) {
    print("big");
}
while (basic.__getboth() > 100) {
    basic.__setboth(0);
}
function usesDefault(a = basic.__getboth()) {
    return a;
}
print(usesDefault());
