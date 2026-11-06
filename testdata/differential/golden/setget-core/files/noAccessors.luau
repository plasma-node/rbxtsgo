-- Compiled with roblox-ts v3.0.0
--[[
	
	 * This file has no accessors. JS reprints it anyway.
	 *        These lines get new indents.
	 
]]
-- numeric spelling
print(1000, 0x1F, 0X1f, 0b101, 0B101, 15, 1e3, 1E3, 1.50, .5, 5., 10.01)
local big = 65535
-- string spelling
print("single", "double", `template`, "it\'s", 'say \"hi\"', "\x41B\u{43}", "é日本語😀")
print(`a{big}b`, `multi\
line`)
print([[\
continued]])
-- blank lines and semicolons
local a = 1
local b = 2
print(a, b)
-- comments in odd places
local o = {
	x = 1,
	y = 2,
}
local arr = { 1, 2 }
print(o, arr)
local function f()
	-- only a comment 
end
f()
local Empty
do
	Empty = setmetatable({}, {
		__tostring = function()
			return "Empty"
		end,
	})
	Empty.__index = Empty
	function Empty.new(...)
		local self = setmetatable({}, Empty)
		return self:constructor(...) or self
	end
	function Empty:constructor()
	end
end
print(Empty.new())
repeat
	if a == 1 then
		-- in a case
		print(1)
		break
	end
	print(2)
until true
for _, v in arr do
	print(v)
end
local fn = function(x)
	return x + 1
end
print(fn(1))
local t = a + b
print(t)
-- parentheses stay
print((1 + 2) * 3, a, b)
print(a + 1, a)
print(arr[1], o.x, o.y)
-- types go through the printer too
print(typeof(u))
-- last comment
return nil
