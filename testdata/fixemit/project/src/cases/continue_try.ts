// A continue that goes through a try as well as a switch. Upstream PR #3125 leaves these with the old bug: its snapshot
// "avoids continue flags for try statements inside a case" still has a bare continue inside the repeat.

function tryInsideCase() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 2:
				try {
					visited.push("try");
					continue;
				} finally {
					visited.push("finally");
				}
		}
		visited.push(`after ${value}`);
	}
	print("try inside case", visited.join(","));
}

function catchInsideCase() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3]) {
		switch (value) {
			case 2:
				try {
					throw "x";
				} catch (e) {
					visited.push("catch");
					continue;
				}
		}
		visited.push(`after ${value}`);
	}
	print("catch inside case", visited.join(","));
}

function switchInsideTry() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3]) {
		try {
			switch (value) {
				case 2:
					visited.push("case");
					continue;
			}
			visited.push(`in try ${value}`);
		} finally {
			visited.push(`finally ${value}`);
		}
		visited.push(`after ${value}`);
	}
	print("switch inside try", visited.join(","));
}

function tryInSwitchInTry() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3]) {
		try {
			switch (value) {
				case 2:
					try {
						continue;
					} finally {
						visited.push("inner finally");
					}
			}
			visited.push(`in try ${value}`);
		} finally {
			visited.push(`outer finally ${value}`);
		}
		visited.push(`after ${value}`);
	}
	print("try in switch in try", visited.join(","));
}

function tryInNestedSwitch() {
	const visited = new Array<string>();
	for (const value of [1, 2]) {
		switch (value) {
			case 2:
				switch (value) {
					case 2:
						try {
							continue;
						} catch (e) {
							visited.push("not reached");
						}
				}
				visited.push("outer");
		}
		visited.push(`after ${value}`);
	}
	print("try in nested switch", visited.join(","));
}

function writeBackThroughTry() {
	const visited = new Array<number>();
	let guard = 0;
	for (let i = 0; i < 5; i++) {
		guard += 1;
		if (guard > 20) {
			break;
		}
		switch (i) {
			case 1:
				try {
					i += 1;
					continue;
				} finally {
					visited.push(-i);
				}
		}
		visited.push(i);
	}
	print("write back through try", visited.join(","), guard);
}

function whileThroughTry() {
	const visited = new Array<number>();
	let count = 0;
	while (count < 3) {
		count += 1;
		switch (count) {
			case 2:
				try {
					continue;
				} catch (e) {}
		}
		visited.push(count);
	}
	print("while through try", visited.join(","));
}

// These work without the fix, and have to keep working.
function breakThroughTry() {
	const visited = new Array<string>();
	for (const value of [1, 2]) {
		switch (value) {
			case 1:
				try {
					break;
				} finally {
					visited.push("finally");
				}
		}
		visited.push(`after ${value}`);
	}
	print("break through try", visited.join(","));
}

function returnThroughTry() {
	for (const value of [1, 2]) {
		switch (value) {
			case 2:
				try {
					return value * 10;
				} finally {
					print("return finally");
				}
		}
	}
	return 0;
}

function breakAndContinueInTry() {
	const visited = new Array<string>();
	for (const value of [1, 2, 3, 4]) {
		switch (value) {
			case 2:
			case 3:
				try {
					if (value === 2) {
						continue;
					}
					break;
				} finally {
					visited.push(`finally ${value}`);
				}
		}
		visited.push(`after ${value}`);
	}
	print("break and continue", visited.join(","));
}

function tryAroundLoop() {
	const visited = new Array<string>();
	try {
		for (const value of [1, 2]) {
			switch (value) {
				case 1:
					continue;
			}
			visited.push(`after ${value}`);
		}
	} finally {
		visited.push("finally");
	}
	print("try around loop", visited.join(","));
}

tryInsideCase();
catchInsideCase();
switchInsideTry();
tryInSwitchInTry();
tryInNestedSwitch();
writeBackThroughTry();
whileThroughTry();
breakThroughTry();
print("return through try", returnThroughTry());
breakAndContinueInTry();
tryAroundLoop();

export {};
