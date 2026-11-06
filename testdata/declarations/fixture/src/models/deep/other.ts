export interface Other {
	other: string;
}
export const other: Other = { other: "x" };
export function parent() {
	return import("../shapes");
}
