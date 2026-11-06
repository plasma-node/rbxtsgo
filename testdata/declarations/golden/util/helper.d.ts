/**
 * A documented function. Comments stay in the declaration.
 * @param value the value
 */
export declare function helper(value: number): number;
export declare const answer = 42;
export declare const text = "quoted 'single' and \"double\"";
export declare const negative = -1.5;
export declare const big = 1e+21;
export declare const hex = 16;
export declare let mutable: string;
export declare const tuple: readonly [1, "two", true];
export declare const nested: {
    a: {
        b: {
            c: number[];
        };
    };
    "quoted-key": number;
};
export declare function defaulted(first?: number, second?: string, ...rest: Array<boolean>): readonly [number, string, boolean[]];
export declare function generator(): Generator<number, void, unknown>;
export type Union = "b" | "a" | 3 | undefined | Instance | Vector3;
export declare const unionValue: Union | number;
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
export declare enum Regular {
    A = 0,
    B = 5,
    C = "c"
}
export declare const enum Inlined {
    X = 1,
    Y = 4
}
export declare abstract class Shapely<T extends object = {}> {
    readonly value: T;
    private other?;
    static count: number;
    protected abstract area(): number;
    private secret;
    constructor(value: T, other?: string | undefined);
    static make(): string;
}
export declare function overloaded(value: string): string;
export declare function overloaded(value: number): number;
export declare namespace Space {
    const inner = 1;
    type InnerType = typeof inner;
    function fn(): number;
}
