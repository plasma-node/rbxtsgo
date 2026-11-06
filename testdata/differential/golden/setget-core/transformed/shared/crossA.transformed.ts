export class Shared {
    private inner = 0;
    public __getdata(): number {
        return this.inner;
    }
    public __setdata(val: number) {
        this.inner = val;
    }
}
export const sharedInstance = new Shared();
export const literalExport = {
    __gety() {
        return 1;
    },
};
export default class Fallback extends Shared {
    public __getextra() {
        return this.__getdata();
    }
}
