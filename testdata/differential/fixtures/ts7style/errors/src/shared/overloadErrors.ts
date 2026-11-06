// TS 5.5 lists every overload at the call. TS 7 reports only the last overload, at the failing argument.
declare function pick(source: string, cb: (v: string) => string): void;
declare function pick(source: number, cb: (v: number) => number): void;
pick("a", (v) => 1);
export {};
