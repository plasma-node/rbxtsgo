export class Shared {
	private inner = 0;
	public get data(): number {
		return this.inner;
	}
	public set data(val: number) {
		this.inner = val;
	}
}

export const sharedInstance = new Shared();

export const literalExport = {
	get y() {
		return 1;
	},
};

export default class Fallback extends Shared {
	public get extra() {
		return this.data;
	}
}
