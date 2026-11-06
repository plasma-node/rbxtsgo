// Sorted before every other fixture that mentions assert, so nothing has created the type of `assert` yet when JS
// resolves this union. The plain function's type comes first in JS, and no macro applies.
declare const plainFirst: ((value: unknown, message?: string) => void) | typeof assert;
plainFirst(true);
