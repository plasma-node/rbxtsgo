// Lone surrogates and non-ASCII text in strings, comments and templates. JS writes U+FFFD for a lone surrogate.
export const loneHigh = "\uD83D";
export const loneLow = "\uDE00";
export const twoLone = "\uD83D\uD83D";
export const pair = "😀";
export const plain = "é 日本語 😀";
// é 日本語 😀 in a comment
/* 😀 in a block */
export const size = "日本語".size();
export const template = `😀 ${plain} é`;
export const escapes = "\n\t\\\"'é\x41\0";
export const multi = "a\
b";
