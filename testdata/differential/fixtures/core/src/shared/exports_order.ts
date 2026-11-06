// The order of a module's exports is the binder's: function declarations first, then source order.
export const first = 1;
export function hoistedLast() {
	return 1;
}
export class Third {}
const local1 = 2;
const local2 = 3;
export { local2 as renamedSecond, local1 };
export const { destructuredB, destructuredA } = { destructuredB: 1, destructuredA: 2 };
export enum Late {
	A,
}
export namespace Space {
	export const inner2 = 1;
	export function innerHoisted() {}
	export const inner1 = 2;
}
export function hoistedFirst() {
	return 2;
}
export default class {}
export type NotAValue = number;
export interface AlsoNot {}
export const aliasTwice = 1;
export { aliasTwice as aliasA, aliasTwice as aliasB };
