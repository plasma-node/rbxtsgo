-- Compiled with roblox-ts v3.0.0
local TS = _G[script]
local exports = {}
--[[
	*
	 * A documented function. Comments stay in the declaration.
	 * @param value the value
	 
]]
local function helper(value)
	return value + 1
end
-- Not a doc comment, so TS drops it.
local answer = 42
local text = [[quoted 'single' and \"double\"]]
local negative = -1.5
local big = 1e21
local hex = 0x10
exports.mutable = "changes"
local tuple = { 1, "two", true }
local nested = {
	a = {
		b = {
			c = { 1, 2, 3 },
		},
	},
	["quoted-key"] = 1,
}
local function defaulted(first, second, ...)
	if first == nil then
		first = 1
	end
	if second == nil then
		second = "x"
	end
	local rest = { ... }
	return { first, second, rest }
end
local function generator()
	return TS.generator(function()
		coroutine.yield(1)
	end)
end
local unionValue = 1
local Regular
do
	local _inverse = {}
	Regular = setmetatable({}, {
		__index = _inverse,
	})
	Regular.A = 0
	_inverse[0] = "A"
	Regular.B = 5
	_inverse[5] = "B"
	Regular.C = "c"
end
local Shapely
do
	Shapely = {}
	function Shapely:constructor(value, other)
		self.value = value
		self.other = other
		self.secret = 1
	end
	function Shapely:make()
		return "made"
	end
	Shapely.count = 0
end
local function overloaded(value)
	return value
end
local Space = {}
do
	local _container = Space
	local inner = 1
	_container.inner = inner
	local function fn()
		return inner
	end
	_container.fn = fn
end
exports.helper = helper
exports.defaulted = defaulted
exports.generator = generator
exports.overloaded = overloaded
exports.answer = answer
exports.text = text
exports.negative = negative
exports.big = big
exports.hex = hex
exports.tuple = tuple
exports.nested = nested
exports.unionValue = unionValue
exports.Regular = Regular
exports.Shapely = Shapely
exports.Space = Space
return exports
