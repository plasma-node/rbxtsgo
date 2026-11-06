import { t } from "@rbxts/t";
import Object from "@rbxts/object-utils";
import Sift from "@rbxts/sift";
import SiftArray from "@rbxts/sift/out/Array";
import { Dictionary, List } from "@rbxts/llama";
import { Option, Result, Vec, HashMap } from "@rbxts/rust-classes";
import StringUtils from "@rbxts/string-utils";

const isPoint = t.interface({ x: t.number, y: t.number, tags: t.array(t.string) });
type Point = t.static<typeof isPoint>;
const isShape = t.union(t.literal("circle"), t.literal("square"));
const checkOptional = t.optional(t.strictInterface({ id: t.integer, name: t.string }));

export function validate(value: unknown): Point | undefined {
	if (isPoint(value)) return value;
	const ok = isShape(value);
	const message = "bad shape";
	if (!ok) warn(message);
	return undefined;
}

export function objects() {
	const original = { a: 1, b: 2, c: 3 };
	const keys = Object.keys(original);
	const values = Object.values(original);
	const entries = Object.entries(original);
	const merged = Object.assign({}, original, { d: 4 });
	const copied = Object.deepCopy(original);
	return { keys, values, entries, merged, copied, optional: checkOptional(undefined) };
}

export function sift() {
	const list = [1, 2, 3, 4, 5];
	const doubled = Sift.Array.map(list, (n) => n * 2);
	const filtered = SiftArray.filter(list, (n) => n % 2 === 0);
	const dict = Sift.Dictionary.merge({ a: 1 }, { b: 2 });
	const set = Sift.Set.add(Sift.Set.fromArray(["x"]), "y");
	const removed = Sift.Dictionary.set({ a: 1, b: 2 }, "b", Sift.None);
	return { doubled, filtered, dict, set, removed };
}

export function llama() {
	const list = List.push([1, 2], 3);
	const dict = Dictionary.set<{ a: number; b?: number }, "b", number>({ a: 1 }, "b", 2);
	return { list, dict };
}

export function rust() {
	const opt = Option.some(5).map((n) => n + 1);
	const res: Result<number, string> = Result.ok(1);
	const vec = Vec.withCapacity<number>(3);
	const map = HashMap.empty<string, number>();
	map.insert("a", 1);
	const text = StringUtils.trim("  padded ").upper();
	return { value: opt.unwrapOr(0), ok: res.isOk(), len: vec.len(), got: map.get("a").unwrapOr(0), text };
}
