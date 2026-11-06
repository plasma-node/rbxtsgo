-- Compiled with roblox-ts v3.0.0
--[[
	*
	 * File header.
	 *     Indented line in a block comment.
	 
]]
-- A line comment before the class.
local Commented
do
	Commented = setmetatable({}, {
		__tostring = function()
			return "Commented"
		end,
	})
	Commented.__index = Commented
	function Commented.new(...)
		local self = setmetatable({}, Commented)
		return self:constructor(...) or self
	end
	function Commented:constructor()
		self.inner = 1
	end
	function Commented:__getvalue()
		-- inside a getter
		return self.inner
		-- end of a getter
	end
	function Commented:__setvalue(val)
		-- inside a setter 
		self.inner = val
	end
	function Commented:__getshared()
		return 1
	end
	function Commented:__getspaced()
		return 2
	end
end
local c = Commented.new()
-- before a set
c:__setvalue(1)
-- before a read
print(c:__getvalue())
-- before a compound
c:__setvalue(c:__getvalue() + 1)
-- before a postfix 
c:__setvalue(c:__getvalue() + 1)
print(c:__getvalue())
print(c:__getvalue())
print(c:__getvalue())
c:__setvalue(3)
local literal = {
	__getx = function(self)
		return 1
	end,
	__setx = function(self, v)
		print(v)
	end,
}
print(literal:__getx())
local function multiline(a, b)
	-- only comment before return
	return a + b
end
print(multiline(1, 2))
if c:__getvalue() > 0 then
	-- only a comment in this block
end
--[[
	
	    Multi-line comment,
	        indented here,
	    at the end of the file.
	
]]
return nil
