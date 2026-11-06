declare function pick(source: string, cb: (v: string) => string): void;
declare function pick(source: number, cb: (v: number) => number): void;
pick("a", (v) => 1);
export {};
