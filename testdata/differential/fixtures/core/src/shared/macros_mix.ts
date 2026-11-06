const arr = [3, 1, 2];
const readonlyArr: ReadonlyArray<number> = arr;
const set = new Set<number>([1]);
const map = new Map<string, number>([["a", 1]]);
const s = "héllo 日本語";
export const a1 = arr.size() + readonlyArr.size() + set.size() + map.size() + s.size();
arr.push(4, 5);
arr.unshift(0);
export const a2 = arr.pop();
export const a3 = arr.shift();
export const a4 = arr.map(v => v * 2).filter(v => v > 1);
export const a5 = arr.reduce((acc, v) => acc + v, 0);
export const a6 = arr.includes(1) && arr.indexOf(2) > 0;
export const a7 = arr.join(", ");
arr.sort((x, y) => x < y);
arr.forEach(v => print(v));
export const a8 = arr.find(v => v > 1);
export const a9 = arr.findIndex(v => v > 1);
export const a10 = arr.every(v => v > 0) || arr.some(v => v > 0);
arr.insert(1, 2);
arr.remove(0);
arr.unorderedRemove(0);
arr.clear();
set.add(1).add(2);
set.delete(1);
set.forEach(v => print(v));
export const a11 = set.has(2);
map.set("x", 1).set("y", 2);
map.delete("x");
map.forEach((v, k) => print(k, v));
export const a12 = map.has("y") ? map.get("y") : 0;
export const a13 = s.sub(1, 2) + s.upper() + s.lower() + s.rep(2) + s.reverse();
export const a14 = s.byte(1);
export const a15 = s.find("l");
export const a16 = s.split(" ");
export const a17 = s.gsub("l", "L");
export const a18 = new Array<number>(3);
export const a19 = new Set([1, 2]);
export const a20 = new Map([["k", 1]]);
export const a21 = new Vector3(1, 2, 3).add(new Vector3(1, 1, 1)).mul(2);
export const a22 = identity<number>(1);
for (const n of $range(1, 3)) {
	print(n);
}
function tup(): LuaTuple<[number, string]> {
	return $tuple(1, "a");
}
const [t1, t2] = tup();
export { t1, t2 };
