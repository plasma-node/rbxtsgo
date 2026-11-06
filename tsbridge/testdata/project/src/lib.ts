export const usedValue = 1;
export const unusedValue = 2;
export interface OnlyType {
	a: number;
}
export class UsedAsTypeOnly {
	b = 1;
}
export default function defaultExport() {
	return 3;
}
export const namedInDefaultFile = 4;
