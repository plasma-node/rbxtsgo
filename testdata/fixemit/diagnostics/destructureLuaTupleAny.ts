declare const iterator: IterableFunction<LuaTuple<any>>;
for (const [first] of iterator) {
	print(first);
}
