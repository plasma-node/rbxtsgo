// Reached through a bare specifier ("shared/util"), so it depends on baseUrl-style resolution.
export enum Team {
	Red,
	Blue = 10,
	Green = "green",
}

export const enum Flag {
	Off = 0,
	On = 1,
}

export interface Point {
	x: number;
	y: number;
}

export type Pair = [name: string, score: number];

export function divmod(a: number, b: number): LuaTuple<[number, number]> {
	return $tuple(math.floor(a / b), a % b);
}

export function firstOf<T extends defined>(items: ReadonlyArray<T>): T | undefined {
	return items[0];
}

export class Counter<T extends defined> {
	protected readonly seen = new Set<T>();

	public add(item: T) {
		this.seen.add(item);
		return this.seen.size();
	}
}

export class NamedCounter extends Counter<string> {
	public constructor(public readonly name: string) {
		super();
	}
}
