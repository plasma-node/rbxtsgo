-- Compiled with roblox-ts v3.0.0
local Value
do
	Value = setmetatable({}, {
		__tostring = function()
			return "Value"
		end,
	})
	Value.__index = Value
	function Value.new(...)
		local self = setmetatable({}, Value)
		return self:constructor(...) or self
	end
	function Value:constructor()
		self.value = 45
	end
	Value.default = 49
end
return Value
