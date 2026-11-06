-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local Roact = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "roact", "src", "ts")
local Base
do
	Base = {}
	function Base:constructor(name, size)
		if size == nil then
			size = 2
		end
		self.name = name
		self.size = size
		self.field = 1
	end
	function Base:greet()
		return "hi " .. self.name
	end
	function Base:make()
		local _original = Base.count
		Base.count += 1
		return _original
	end
	function Base:toString()
		return self.name
	end
	function Base:__tostring()
		return self:toString()
	end
	Base.count = 0
end
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
		self.extra = "x"
	end
	function Child:act()
		print(super.greet(self))
	end
	function Child:__tostring()
		return self:toString()
	end
end
local function dec(c)
end
local function propDec(target, key)
end
local Decorated
do
	Decorated = setmetatable({}, {
		__tostring = function()
			return "Decorated"
		end,
	})
	Decorated.__index = Decorated
	function Decorated.new(...)
		local self = setmetatable({}, Decorated)
		return self:constructor(...) or self
	end
	function Decorated:constructor()
		self.prop = 1
	end
	propDec(Decorated, "prop")
	Decorated = dec(Decorated) or Decorated
end
local _exp = Roact.jsx("textlabel", {
	Text = "a &amp; 😀 b",
})
local _exp_1 = Roact.jsx("textbutton", {
	Text = "x",
})
local _exp_2 = { 1, 2 }
-- ▼ ReadonlyArray.map ▼
local _newValue = table.create(#_exp_2)
local _callback = function(n)
	return Roact.jsx("textlabel", {
		Text = `{n}`,
	})
end
for _k, _v in _exp_2 do
	_newValue[_k] = _callback(_v, _k - 1, _exp_2)
end
-- ▲ ReadonlyArray.map ▲
local Elt = (Roact.jsx("frame", {
	Size = UDim2.new(0, 1, 0, 1),
	Key = "k",
	Event = {
		MouseEnter = function()
			return print("&amp; é")
		end,
	},
}, _exp, _exp_1, _newValue))
return {
	Child = Child,
	Decorated = Decorated,
	Elt = Elt,
}
