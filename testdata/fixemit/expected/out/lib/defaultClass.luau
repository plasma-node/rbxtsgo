-- Compiled with roblox-ts v3.0.0
local DefaultClass
do
	DefaultClass = setmetatable({}, {
		__tostring = function()
			return "DefaultClass"
		end,
	})
	DefaultClass.__index = DefaultClass
	function DefaultClass.new(...)
		local self = setmetatable({}, DefaultClass)
		return self:constructor(...) or self
	end
	function DefaultClass:constructor()
		self.value = 44
	end
end
return {
	default = DefaultClass,
}
