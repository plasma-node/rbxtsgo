// Offsets in this file are UTF-16 units in JS and bytes in Go, so the probe drops them here. café 日本語 😀
export const accent = "é";
export const cjk = "日本語";
export const emoji = "😀";
export const escaped = "é\u{1F600}";
export const loneSurrogate = "\uD83D";
export const café = 1;
export const tmpl = `日本語${accent}😀`;
