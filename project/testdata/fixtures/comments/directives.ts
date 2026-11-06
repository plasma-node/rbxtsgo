// @ts-ignore
const a: number = "x";

  // @ts-expect-error something
const b: number = "y";

/* @ts-ignore */
const c: number = "z";

const d = 1; // not a directive
export { a, b, c, d };
