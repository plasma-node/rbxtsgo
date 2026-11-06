/**
 * A documented function. Comments stay in the declaration.
 * @param value the value
 */
export function helper(value: number): number {
	return value + 1;
}

// Not a doc comment, so TS drops it.
export const answer = 42;
export const text = "quoted 'single' and \"double\"";
export const negative = -1.5;
export const big = 1e21;
export const hex = 0x10;
export let mutable = "changes";
export const tuple = [1, "two", true] as const;
export const nested = { a: { b: { c: [1, 2, 3] } }, "quoted-key": 1 };
export function defaulted(first = 1, second = "x", ...rest: Array<boolean>) {
	return [first, second, rest] as const;
}
export function* generator() {
	yield 1;
}
export type Union = "b" | "a" | 3 | undefined | Instance | Vector3;
export const unionValue: Union | number = 1 as number;
export interface Merged {
	one: number;
}
export interface Merged {
	two?: string;
	readonly three: Array<Map<string, Set<number>>>;
	method(this: Merged, value?: number): void;
	new (value: number): Merged;
	(call: string): void;
	[key: string]: unknown;
}
export enum Regular {
	A,
	B = 5,
	C = "c",
}
export const enum Inlined {
	X = 1,
	Y = X << 2,
}
export abstract class Shapely<T extends object = {}> {
	public static count = 0;
	protected abstract area(): number;
	private secret = 1;
	public constructor(public readonly value: T, private other?: string) {}
	public static make() {
		return "made";
	}
}
export function overloaded(value: string): string;
export function overloaded(value: number): number;
export function overloaded(value: string | number) {
	return value;
}
export namespace Space {
	export const inner = 1;
	export type InnerType = typeof inner;
	export function fn() {
		return inner;
	}
}
