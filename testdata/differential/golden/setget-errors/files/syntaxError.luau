-- Compiled with roblox-ts v3.0.0
local C
do
	C = setmetatable({}, {
		__tostring = function()
			return "C"
		end,
	})
	C.__index = C
	function C.new(...)
		local self = setmetatable({}, C)
		return self:constructor(...) or self
	end
	function C:constructor()
	end
	function C:__getv()
		return 1
	end
end
local c = C.new()
print(c:__getv())
return nil
