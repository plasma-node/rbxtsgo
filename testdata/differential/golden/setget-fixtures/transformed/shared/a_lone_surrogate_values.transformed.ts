// A lone surrogate reaches a Luau string through a decoded value: tagged templates and enum and property names.
declare function tag(strings: TemplateStringsArray, ...values: Array<unknown>): void;
tag `\uD83D`;
tag `\uD83D\uD83D ${1} \uDE00`;
tag `😀`;
enum WithLone {
    A = "\uD83D"
}
print(WithLone.A);
