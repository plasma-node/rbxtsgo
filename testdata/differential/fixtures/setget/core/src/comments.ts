/**
 * File header.
 *     Indented line in a block comment.
 */

// A line comment before the class.
class Commented {
	private inner = 1; // trailing on a field

	// before a getter
	/** doc for the getter */
	public get value(): number {
		// inside a getter
		return this.inner; // trailing in a getter
		// end of a getter
	}

	/* before a setter */
	public set value(val: number) /* after the signature */ {
		/* inside a setter */
		this.inner = val;
	}

	// before a static accessor
	public static get shared(): number /* return type */ {
		return 1;
	}

	/* before a modifier */ public /* after a modifier */ get /* after get */ spaced /* after the name */ () {
		return 2;
	}

	// after the last accessor
}

const c = new Commented();
// before a set
c.value = 1; // after a set
// before a read
print(c.value); // after a read
// before a compound
c.value += /* inside a compound */ 1;
/* before a postfix */ c.value++; /* after a postfix */
print(/* before */ c.value /* after */);
print(c /* mid */ .value);
print(c
	// on its own line
	.value);
c /* a */ . /* b */ value /* c */ = /* d */ 3 /* e */;

const literal = {
	// before a literal getter
	get x() {
		return 1; // in a literal getter
	},
	/* before a literal setter */
	set x(v: number) {
		print(v);
	},
	// after the literal accessors
};
print(literal.x);

function multiline(
	// before a parameter
	a: number, // after a parameter
	/* before b */ b: number,
) {
	// only comment before return
	return a + b;
}
print(multiline(1, 2));

if (c.value > 0) {
	// only a comment in this block
}

/*
	Multi-line comment,
		indented here,
	at the end of the file.
*/
