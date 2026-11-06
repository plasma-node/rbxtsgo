// A global augmentation, which real @rbxts packages also do. It forces the checker
// to merge the Array global, so each checker gets its own clone of the symbol.
declare global {
	interface Array<T> {
		spikeAugmented(): T;
	}
}

export {};
