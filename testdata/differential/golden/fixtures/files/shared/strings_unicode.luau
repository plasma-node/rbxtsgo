-- Compiled with roblox-ts v3.0.0
-- Lone surrogates and non-ASCII text in strings, comments and templates. JS writes U+FFFD for a lone surrogate.
local loneHigh = "\uD83D"
local loneLow = "\uDE00"
local twoLone = "\uD83D\uD83D"
local pair = "😀"
local plain = "é 日本語 😀"
-- é 日本語 😀 in a comment
-- 😀 in a block 
local size = #"日本語"
local template = `😀 {plain} é`
local escapes = [[\n\t\\\"'é\x41\0]]
local multi = [[a\
b]]
return {
	loneHigh = loneHigh,
	loneLow = loneLow,
	twoLone = twoLone,
	pair = pair,
	plain = plain,
	size = size,
	template = template,
	escapes = escapes,
	multi = multi,
}
