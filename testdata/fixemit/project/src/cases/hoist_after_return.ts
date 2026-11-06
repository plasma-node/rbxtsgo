// Function declarations after a final statement. Issues #2847 and #2098, open upstream with no fix, so nothing here is
// ported. In JS the function exists from the start of its block, and sees every binding in the block.

// #2098
function calledByTheReturn() {
	return bar();
	function bar() {
		return "Hello, world!";
	}
}

// #2847
function assignsAnOuterLocal() {
	let variable = 0;

	SetVariable(2);
	return variable;

	function SetVariable(value: number) {
		variable = value;
	}
}

function calledBeforeLocalsItReads() {
	const before = readLater();
	const later = 5;
	const after = readLater();
	return `${before} ${after} ${later}`;

	function readLater() {
		return typeIs(later, "nil") ? "unset" : "set";
	}
}

function readsDestructuredLocal() {
	const [a, b] = [1, 2];
	const { c } = { c: 3 };
	return sum();

	function sum() {
		return a + b + c;
	}
}

function mutualRecursion() {
	return isEven(4) && !isEven(3);

	function isEven(n: number): boolean {
		return n === 0 ? true : isOdd(n - 1);
	}
	function isOdd(n: number): boolean {
		return n === 0 ? false : isEven(n - 1);
	}
}

function callsAFunctionBeforeTheReturn() {
	function early() {
		return late() + 1;
	}
	return early();
	function late() {
		return 41;
	}
}

function readsFunctionAsAValue() {
	const callback = valued;
	return callback();
	function valued() {
		return "value";
	}
}

function laterStatementsStayDropped() {
	const order = new Array<string>();
	order.push("before");
	const result = visible();
	return `${order.join(",")} ${result}`;
	order.push("after the return");
	function visible() {
		order.push("in function");
		return "done";
	}
	order.push("after the function");
}

function classAfterReturnStaysDropped() {
	return helper();
	class NotHoisted {}
	function helper() {
		return "class dropped";
	}
}

function insideIfBlock(flag: boolean) {
	if (flag) {
		return inner();
		function inner() {
			return "in block";
		}
	}
	return "skipped";
}

function afterContinueInLoop() {
	const visited = new Array<number>();
	for (const value of [1, 2]) {
		visited.push(double(value));
		continue;
		function double(n: number) {
			return n * 2;
		}
	}
	return visited.join(",");
}

function afterBreakInLoop() {
	let result = 0;
	while (true) {
		result = three();
		break;
		function three() {
			return 3;
		}
	}
	return result;
}

function afterBreakInCase(value: number) {
	switch (value) {
		case 1:
			print("case", inCase());
			break;
			function inCase() {
				return "in case";
			}
	}
	return value;
}

function overloads() {
	return overloaded(1);
	function overloaded(a: number): number;
	function overloaded(a: string): string;
	function overloaded(a: number | string) {
		return a;
	}
}

function capturesParameter(parameter: number) {
	return getParameter();
	function getParameter() {
		return parameter;
	}
}

function generatorBody() {
	function* numbers() {
		yield one();
		return 2;
		function one() {
			return 1;
		}
	}
	const iterator = numbers();
	return iterator.next().value;
}

function nestedAfterReturn() {
	return outer();
	function outer() {
		return inner();
		function inner() {
			return "nested";
		}
	}
}

function usedLaterThanTheCall() {
	let count = 0;
	increment();
	increment();
	return count;
	function increment() {
		count += 1;
	}
}

print("#2098", calledByTheReturn());
print("#2847", assignsAnOuterLocal());
print("destructured", readsDestructuredLocal());
print("mutual", mutualRecursion());
print("before the return", callsAFunctionBeforeTheReturn());
print("as a value", readsFunctionAsAValue());
print("dropped", laterStatementsStayDropped());
print("class", classAfterReturnStaysDropped());
print("if", insideIfBlock(true), insideIfBlock(false));
print("continue", afterContinueInLoop());
print("break", afterBreakInLoop());
print("case", afterBreakInCase(1));
print("overloads", overloads());
print("parameter", capturesParameter(7));
print("generator", generatorBody());
print("nested", nestedAfterReturn());
print("count", usedLaterThanTheCall());

export = calledBeforeLocalsItReads;
