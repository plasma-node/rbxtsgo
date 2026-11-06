-- Compiled with roblox-ts v3.0.0
local Shape
do
	Shape = {}
	function Shape:constructor()
		self.size = 1
	end
	function Shape:__getdoubled()
		return self.size * 2
	end
	function Shape:__setdoubled(val)
		self.size = val / 2
	end
	function Shape:__getlabel()
		return "shape"
	end
end
local Square
do
	local super = Shape
	Square = setmetatable({}, {
		__tostring = function()
			return "Square"
		end,
		__index = super,
	})
	Square.__index = Square
	function Square.new(...)
		local self = setmetatable({}, Square)
		return self:constructor(...) or self
	end
	function Square:constructor(...)
		super.constructor(self, ...)
	end
	function Square:__getarea()
		return self.size * self.size
	end
	function Square:__getlabel()
		return "square " .. super.__getlabel(self)
	end
	function Square:grow()
		super.__setdoubled(self, 10)
		self:__setdoubled(self:__getdoubled() + 1)
		return super.__getdoubled(self)
	end
end
local Tiny
do
	local super = Square
	Tiny = setmetatable({}, {
		__tostring = function()
			return "Tiny"
		end,
		__index = super,
	})
	Tiny.__index = Tiny
	function Tiny.new(...)
		local self = setmetatable({}, Tiny)
		return self:constructor(...) or self
	end
	function Tiny:constructor(...)
		super.constructor(self, ...)
	end
	function Tiny:__getdoubled()
		return super.__getdoubled(self) + 1
	end
	function Tiny:__setdoubled(val)
		super.__setdoubled(self, val - 1)
	end
end
local GetInBaseSetHere
do
	local super = Shape
	GetInBaseSetHere = setmetatable({}, {
		__tostring = function()
			return "GetInBaseSetHere"
		end,
		__index = super,
	})
	GetInBaseSetHere.__index = GetInBaseSetHere
	function GetInBaseSetHere.new(...)
		local self = setmetatable({}, GetInBaseSetHere)
		return self:constructor(...) or self
	end
	function GetInBaseSetHere:constructor(...)
		super.constructor(self, ...)
	end
	function GetInBaseSetHere:__getarea()
		return 0
	end
	function GetInBaseSetHere:__setlabel(val)
		print(val)
	end
end
local square = Square.new()
print(square:__getarea(), square:__getlabel(), square:grow())
local tiny = Tiny.new()
tiny:__setdoubled(4)
print(tiny:__getdoubled(), tiny:__getarea())
local asShape = tiny
print(asShape:__getarea(), asShape:__getlabel())
asShape:__setdoubled(8)
local other = GetInBaseSetHere.new()
other:__setlabel("x")
local Named
do
	Named = setmetatable({}, {
		__tostring = function()
			return "Named"
		end,
	})
	Named.__index = Named
	function Named.new(...)
		local self = setmetatable({}, Named)
		return self:constructor(...) or self
	end
	function Named:constructor()
		self.inner = "n"
	end
	function Named:__getname()
		return self.inner
	end
	function Named:__setname(value)
		self.inner = value
	end
end
local named = Named.new()
print(named:__getname())
local concrete = Named.new()
concrete:__setname("m")
local Box
do
	Box = setmetatable({}, {
		__tostring = function()
			return "Box"
		end,
	})
	Box.__index = Box
	function Box.new(...)
		local self = setmetatable({}, Box)
		return self:constructor(...) or self
	end
	function Box:constructor(item)
		self.item = item
	end
	function Box:__getvalue()
		return self.item
	end
	function Box:__setvalue(val)
		self.item = val
	end
end
local box = Box.new(1)
box:__setvalue(box:__getvalue() + 2)
print(Box.new("s"):__getvalue(), box:__getvalue())
return nil
