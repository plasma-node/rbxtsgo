import { makeMap } from "shared/viaBaseUrl";

export function sum(values: Iterable<number>) {
	let total = 0;
	for (const value of values) {
		total += value;
	}
	for (const [key, value] of makeMap()) {
		total += value + key.size();
	}
	const set = new Set<number>();
	for (const value of set) {
		total += value;
	}
	const spread = [...set];
	const [first] = spread;
	return total + first + ambientFromTypes();
}
