-- Compiled with roblox-ts v3.0.0
-- Unions whose members have different symbols, because getFirstDefinedSymbol reads the first member.
-- Each case is written in both member orders. See docs/union-order.md.
local ClassWithSize
do
	ClassWithSize = setmetatable({}, {
		__tostring = function()
			return "ClassWithSize"
		end,
	})
	ClassWithSize.__index = ClassWithSize
	function ClassWithSize.new(...)
		local self = setmetatable({}, ClassWithSize)
		return self:constructor(...) or self
	end
	function ClassWithSize:constructor()
	end
	function ClassWithSize:size()
		return 1
	end
end
local a = #macroOrCustom
local b = #customOrMacro
local c = #macroOrClass
local d = #classOrMacro
local e = #tupleOrCustom
local f = #customOrTuple
-- A call macro in a union with a plain function.
assert(true)
assert(true)
-- The same interface declared before and after the array in a union, used as a value and with an optional call.
local _g = optionalUnion
if _g ~= nil then
	_g = #_g
end
local g = _g
local _h = optionalReverse
if _h ~= nil then
	_h = #_h
end
local h = _h
return {
	a = a,
	b = b,
	c = c,
	d = d,
	e = e,
	f = f,
	g = g,
	h = h,
}
