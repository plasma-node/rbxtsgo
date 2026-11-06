-- Compiled with roblox-ts v3.0.0
-- The order of a module's exports is the binder's: function declarations first, then source order.
local first = 1
local function hoistedLast()
	return 1
end
local Third
do
	Third = setmetatable({}, {
		__tostring = function()
			return "Third"
		end,
	})
	Third.__index = Third
	function Third.new(...)
		local self = setmetatable({}, Third)
		return self:constructor(...) or self
	end
	function Third:constructor()
	end
end
local local1 = 2
local local2 = 3
local _binding = {
	destructuredB = 1,
	destructuredA = 2,
}
local destructuredB = _binding.destructuredB
local destructuredA = _binding.destructuredA
local Late
do
	local _inverse = {}
	Late = setmetatable({}, {
		__index = _inverse,
	})
	Late.A = 0
	_inverse[0] = "A"
end
local Space = {}
do
	local _container = Space
	local inner2 = 1
	_container.inner2 = inner2
	local function innerHoisted()
	end
	_container.innerHoisted = innerHoisted
	local inner1 = 2
	_container.inner1 = inner1
end
local function hoistedFirst()
	return 2
end
local default
do
	default = setmetatable({}, {
		__tostring = function()
			return "default"
		end,
	})
	default.__index = default
	function default.new(...)
		local self = setmetatable({}, default)
		return self:constructor(...) or self
	end
	function default:constructor()
	end
end
local aliasTwice = 1
return {
	hoistedLast = hoistedLast,
	hoistedFirst = hoistedFirst,
	first = first,
	Third = Third,
	renamedSecond = local2,
	local1 = local1,
	destructuredB = destructuredB,
	destructuredA = destructuredA,
	Late = Late,
	Space = Space,
	default = default,
	aliasTwice = aliasTwice,
	aliasA = aliasTwice,
	aliasB = aliasTwice,
}
