-- Compiled with roblox-ts v3.0.0
local Thing
do
	Thing = setmetatable({}, {
		__tostring = function()
			return "Thing"
		end,
	})
	Thing.__index = Thing
	function Thing.new(...)
		local self = setmetatable({}, Thing)
		return self:constructor(...) or self
	end
	function Thing:constructor()
		self.field = 1
	end
end
return Thing
