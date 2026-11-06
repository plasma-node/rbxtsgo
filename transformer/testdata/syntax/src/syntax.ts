// A public, from-scratch file that exercises the syntax the transformer handles.
import defaultThing, { helper as renamed, type Shape, Kind } from "./other";
import * as whole from "./other";

// numeric and string literal forms
export const numbers = [0, 1, 1.5, 0.5, 0x10, 0o17, 0b101, 1e3, 1_000, 1e-3, 123456789012345680000, 0.1 + 0.2];
export const strings = ["plain", 'single', "esc\n\t\\\"q", "\x41B\u{43}", `tmpl`, "multi\
line"];
const name = "world";
export const templated = `hello ${name} and ${1 + 1}\n end \${raw}`;
function tag(strs: TemplateStringsArray, ...vals: Array<unknown>) {
	return strs.size() + vals.size();
}
export const tagged = tag`a${1}b\n${2}c`;

// enums
export enum Color {
	Red,
	Green = 5,
	Blue,
	Name = "name",
}
export const enum Flag {
	A = 1,
	B = A << 1,
	C = A | B,
}
export const enumUses = [Color.Red, Color.Blue, Color["Green"], Color.Name, Flag.C, Kind.One];

// classes
abstract class Base<T> {
	public static count = 0;
	protected abstract value: T;
	public constructor(public readonly id: number, private secret?: string) {}
	public static make() {
		return Base.count++;
	}
	public method(this: Base<T>, arg: T) {
		return [arg, this.id] as const;
	}
	public get prop() {
		return this.id;
	}
}
export class Derived extends Base<string> implements Shape {
	protected value = "v";
	public sides = 3;
	public override method(arg: string) {
		return super.method(arg);
	}
	public arrow = () => this.sides;
	static {
		Derived.count = 1;
	}
}
function sealed(target: unknown) {}
@sealed
export class Decorated {
	public x?: number;
}

// namespaces
export namespace Space {
	export const inner = 1;
	export function run() {
		return inner + 1;
	}
	export namespace Deep {
		export const deeper = inner;
	}
}

// functions and generators
export function* gen(n: number): Generator<number, void, undefined> {
	for (let i = 0; i < n; i++) {
		yield i;
	}
}
export async function later(p: Promise<number>) {
	const v = await p;
	return v + 1;
}
export function defaults(a = 1, b?: string, ...rest: Array<number>) {
	return [a, b, rest.size()];
}
const fn = function named(x: number): number {
	return x > 0 ? named(x - 1) : 0;
};

// destructuring and spread
type Pair = [number, string];
export function destructure(pair: Pair, obj: { a: number; b?: { c: string } }, map: Map<string, number>) {
	const [first, second = "x"] = pair;
	const { a, b: { c } = { c: "d" } } = obj;
	let x = 0;
	let y = "";
	[x, y] = pair;
	({ a: x } = obj);
	const copy = { ...obj, extra: 1 };
	const list = [...pair, ...[1, 2]];
	for (const [k, v] of map) {
		print(k, v);
	}
	return [first, second, a, c, x, y, copy, list];
}

// optional chains, nullish, casts
export function chains(o?: { f?: () => number; g: { h?: number }; arr: Array<number> }) {
	const a = o?.f?.();
	const b = o?.g.h ?? 2;
	const c = o!.g.h!;
	const d = (o as unknown as { z: number }).z;
	const e = <number>(<unknown>a);
	const f = { y: 1 } satisfies { y: number };
	let g: number | undefined;
	g ??= 1;
	g ||= 2;
	g &&= 3;
	return [a, b, c, d, e, f, g, typeOf(a), o?.arr[0], o?.["g"]];
}

// statements
export function statements(v: unknown, n: number, set: Set<number>, tuple: LuaTuple<[number, string]>) {
	label: for (const i of set) {
		if (i > 1) continue label;
		else break label;
	}
	switch (n) {
		case 1:
			return "one";
		case 2: {
			break;
		}
		default:
			n++;
	}
	try {
		error("x");
	} catch (e) {
		print(e);
	} finally {
		n--;
	}
	do {
		n += 1;
	} while (n < 3);
	while (n > 0) n--;
	for (const key in { a: 1 }) print(key);
	const [p, q] = tuple;
	const obj: Record<string, number> = {};
	delete obj.k;
	void obj;
	const t = typeIs(v, "string") ? v : undefined;
	const r = classIs(game, "DataModel");
	return [p, q, t, r, $tuple(1, 2)];
}

// types the checker has to work out
export type Mapped<T> = { readonly [K in keyof T]?: T[K] };
export type Cond<T> = T extends string ? "s" : T extends number ? "n" : never;
export interface Callable {
	(a: number): string;
	new (b: string): Callable;
	field: number;
	method(): void;
	[index: number]: string;
}
export function usesTypes(c: Callable, m: Mapped<{ a: 1 }>, u: string | number | undefined, i: { a: 1 } & { b: 2 }, l: "x" | "y", t: readonly [number, ...string[]]) {
	const called = c(1);
	const built = new c("s");
	const indexed = c[0];
	return [called, built.field, indexed, m.a, u, i.a, i.b, l, t[0], t.size(), renamed(), whole.helper, defaultThing];
}
export default class Exported {}
export { fn as exportedFn };
