import { atom, batch, computed, effect, mapped, observe, subscribe, signal, untracked } from "@rbxts/charm";

const count = atom(0);
const doubled = computed(() => count() * 2);
const items = atom<readonly string[]>(["a", "b"]);
const [getName, setName] = signal("x");

export function run() {
	const stop = subscribe(doubled, (value, previous) => print(value, previous));
	const cleanup = effect(() => {
		print(count(), getName());
		return () => print("effect cleanup");
	});
	batch(() => {
		count((n) => n + 1);
		count(10);
		setName("y");
	});
	const lengths = mapped(items, (item, index) => `${index}:${item}`);
	observe(items, (item, index) => {
		print("added", item, index);
		return () => print("removed", item);
	});
	const peeked = untracked(() => count());
	stop();
	cleanup();
	return { lengths, peeked, lenValue: lengths() };
}
