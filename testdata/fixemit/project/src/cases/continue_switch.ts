// Upstream PR #3125's runtime tests, in tests/src/tests/switch.spec.ts, printing what they compare. Issue #2899.

function forOfLoop() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 2:
				visited.push("skipped");
				continue;
		}
		visited.push(`after ${value}`);
	}
	print("for-of", visited.join(","));
}

function whileLoop() {
	const visited = new Array<number>();
	let count = 0;
	while (count < 3) {
		count += 1;
		switch (count) {
			case 2:
				continue;
		}
		visited.push(count);
	}
	print("while", visited.join(","));
}

function nestedSwitch() {
	const visited = new Array<string>();
	for (const value of [1, 2]) {
		switch (value) {
			case 2:
				switch (value) {
					case 2:
						visited.push("inner");
						continue;
				}
				visited.push("outer");
		}
		visited.push(`after ${value}`);
	}
	print("nested switch", visited.join(","));
}

function loopInsideCase() {
	const visited = new Array<string>();
	for (const outer of [1, 2]) {
		switch (outer) {
			case 1:
				for (const inner of [1, 2, 3]) {
					if (inner === 2) {
						continue;
					}
					visited.push(`inner ${inner}`);
				}
				break;
		}
		visited.push(`outer ${outer}`);
	}
	print("loop inside case", visited.join(","));
}

function writeBack() {
	const visited = new Array<number>();
	// writing the loop variable in the body forces the unoptimized loop form, which copies it back every iteration
	let guard = 0;
	for (let i = 0; i < 6; i++) {
		// a dropped write back turns the continue below into an infinite loop
		guard += 1;
		if (guard > 20) {
			break;
		}
		switch (i) {
			case 1:
				continue;
			case 3:
				i += 1;
				break;
		}
		visited.push(i);
	}
	print("write back", visited.join(","), guard);
}

function doWhile() {
	const visited = new Array<number>();
	let count = 0;
	do {
		count += 1;
		switch (count) {
			case 2:
				continue;
		}
		visited.push(count);
	} while (count < 3);
	print("do-while", visited.join(","));
}

function fallthroughAndBreak() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3, 4]) {
		switch (value) {
			case 1:
			case 2:
				if (value === 2) {
					continue;
				}
				visited.push("one or two");
				break;
			case 3:
				break;
			default:
				visited.push("default");
		}
		visited.push(`after ${value}`);
	}
	print("fallthrough", visited.join(","));
}

function multipleContinues() {
	const visited = new Array<number>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 1:
				continue;
			case 2:
				continue;
		}
		visited.push(value);
	}
	print("multiple", visited.join(","));
}

function defaultClause() {
	const visited = new Array<number>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 1:
				break;
			default:
				continue;
		}
		visited.push(value);
	}
	print("default clause", visited.join(","));
}

function noContinue() {
	const visited = new Array<string>();
	for (const value of [1, 2]) {
		switch (value) {
			case 1:
				visited.push("a");
				break;
			default:
				visited.push("b");
		}
		visited.push(`after ${value}`);
	}
	print("no continue", visited.join(","));
}

function closureInLoop() {
	const callbacks = new Array<() => number>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 2:
				continue;
		}
		callbacks.push(() => value);
	}
	const values = new Array<number>();
	for (const callback of callbacks) {
		values.push(callback());
	}
	print("closure", values.join(","));
}

forOfLoop();
whileLoop();
nestedSwitch();
loopInsideCase();
writeBack();
doWhile();
fallthroughAndBreak();
multipleContinues();
defaultClause();
noContinue();
closureInLoop();

export {};
