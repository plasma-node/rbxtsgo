// Issue #2371, in a class this can extend: `this` here is a type JS can't iterate.
class Bag {
	list() {
		for (const value of this as unknown as IterableIterator<number>) {
			print(value);
		}
	}
}
export = Bag;
