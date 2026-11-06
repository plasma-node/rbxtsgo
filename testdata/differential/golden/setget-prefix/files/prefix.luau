-- Compiled with roblox-ts v3.0.0
local P
do
	P = setmetatable({}, {
		__tostring = function()
			return "P"
		end,
	})
	P.__index = P
	function P.new(...)
		local self = setmetatable({}, P)
		return self:constructor(...) or self
	end
	function P:constructor()
		self.inner = 1
	end
	function P:_geta()
		return self.inner
	end
	function P:_seta(val)
		self.inner = val
	end
end
local p = P.new()
p:_seta(1)
p:_seta(p:_geta() + 1)
p:_seta(p:_geta() + 1)
print(p:_geta())
return nil
