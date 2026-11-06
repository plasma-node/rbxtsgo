-- Compiled with roblox-ts v3.0.0
-- Reached through a bare specifier ("shared/util"), so it depends on baseUrl-style resolution.
local Team
do
	local _inverse = {}
	Team = setmetatable({}, {
		__index = _inverse,
	})
	Team.Red = 0
	_inverse[0] = "Red"
	Team.Blue = 10
	_inverse[10] = "Blue"
	Team.Green = "green"
end
local function divmod(a, b)
	return math.floor(a / b), a % b
end
local function firstOf(items)
	return items[1]
end
local Counter
do
	Counter = setmetatable({}, {
		__tostring = function()
			return "Counter"
		end,
	})
	Counter.__index = Counter
	function Counter.new(...)
		local self = setmetatable({}, Counter)
		return self:constructor(...) or self
	end
	function Counter:constructor()
		self.seen = {}
	end
	function Counter:add(item)
		local _seen = self.seen
		local _item = item
		_seen[_item] = true
		-- ▼ ReadonlySet.size ▼
		local _size = 0
		for _ in self.seen do
			_size += 1
		end
		-- ▲ ReadonlySet.size ▲
		return _size
	end
end
local NamedCounter
do
	local super = Counter
	NamedCounter = setmetatable({}, {
		__tostring = function()
			return "NamedCounter"
		end,
		__index = super,
	})
	NamedCounter.__index = NamedCounter
	function NamedCounter.new(...)
		local self = setmetatable({}, NamedCounter)
		return self:constructor(...) or self
	end
	function NamedCounter:constructor(name)
		super.constructor(self)
		self.name = name
	end
end
return {
	divmod = divmod,
	firstOf = firstOf,
	Team = Team,
	Counter = Counter,
	NamedCounter = NamedCounter,
}
