-- Compiled with roblox-ts v3.0.0
local Shared
do
	Shared = setmetatable({}, {
		__tostring = function()
			return "Shared"
		end,
	})
	Shared.__index = Shared
	function Shared.new(...)
		local self = setmetatable({}, Shared)
		return self:constructor(...) or self
	end
	function Shared:constructor()
		self.inner = 0
	end
	function Shared:__getdata()
		return self.inner
	end
	function Shared:__setdata(val)
		self.inner = val
	end
end
local sharedInstance = Shared.new()
local literalExport = {
	__gety = function(self)
		return 1
	end,
}
local Fallback
do
	local super = Shared
	Fallback = setmetatable({}, {
		__tostring = function()
			return "Fallback"
		end,
		__index = super,
	})
	Fallback.__index = Fallback
	function Fallback.new(...)
		local self = setmetatable({}, Fallback)
		return self:constructor(...) or self
	end
	function Fallback:constructor(...)
		super.constructor(self, ...)
	end
	function Fallback:__getextra()
		return self:__getdata()
	end
end
return {
	Shared = Shared,
	sharedInstance = sharedInstance,
	literalExport = literalExport,
	default = Fallback,
}
