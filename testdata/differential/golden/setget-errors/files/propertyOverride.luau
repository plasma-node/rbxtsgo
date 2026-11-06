-- Compiled with roblox-ts v3.0.0
local Base
do
	Base = setmetatable({}, {
		__tostring = function()
			return "Base"
		end,
	})
	Base.__index = Base
	function Base.new(...)
		local self = setmetatable({}, Base)
		return self:constructor(...) or self
	end
	function Base:constructor()
		self.v = 1
	end
end
-- TS itself rejects this before the reprint, and JS accepts it, because the getter is a method by then.
local Child
do
	local super = Base
	Child = setmetatable({}, {
		__tostring = function()
			return "Child"
		end,
		__index = super,
	})
	Child.__index = Child
	function Child.new(...)
		local self = setmetatable({}, Child)
		return self:constructor(...) or self
	end
	function Child:constructor(...)
		super.constructor(self, ...)
	end
	function Child:__getv()
		return 2
	end
end
print(Child.new():__getv())
return nil
