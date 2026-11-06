/**
 * File header.
 *     Indented line in a block comment.
 */
// A line comment before the class.
class Commented {
    private inner = 1; // trailing on a field
    // before a getter
    /** doc for the getter */
    public __getvalue(): number {
        // inside a getter
        return this.inner; // trailing in a getter
        // end of a getter
    }
    /* before a setter */
    public __setvalue(val: number) {
        /* inside a setter */
        this.inner = val;
    }
    // before a static accessor
    public static __getshared(): number /* return type */ {
        return 1;
    }
    /* before a modifier */ public /* after a modifier */ __getspaced() {
        return 2;
    }
}
const c = new Commented();
// before a set
c.__setvalue(1); // after a set
// before a read
print(c.__getvalue()); // after a read
// before a compound
c.__setvalue(c.__getvalue() + 1);
/* before a postfix */ c.__setvalue(c.__getvalue() + 1); /* after a postfix */
print(c.__getvalue /* after */());
print(c /* mid */.__getvalue());
print(c
// on its own line
.__getvalue());
c /* a */.__setvalue(/* d */ 3 /* e */);
const literal = {
    __getx() {
        return 1; // in a literal getter
    },
    __setx(v: number) {
        print(v);
    },
    // after the literal accessors
};
print(literal.__getx());
function multiline(
// before a parameter
a: number, // after a parameter
/* before b */ b: number) {
    // only comment before return
    return a + b;
}
print(multiline(1, 2));
if (c.__getvalue() > 0) {
    // only a comment in this block
}
/*
    Multi-line comment,
        indented here,
    at the end of the file.
*/
