-- Compiled with roblox-ts v3.0.0
local Klass
do
	Klass = setmetatable({}, {
		__tostring = function()
			return "Klass"
		end,
	})
	Klass.__index = Klass
	function Klass.new(...)
		local self = setmetatable({}, Klass)
		return self:constructor(...) or self
	end
	function Klass:constructor()
	end
end
return Klass
