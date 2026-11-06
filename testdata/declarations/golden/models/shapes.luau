-- Compiled with roblox-ts v3.0.0
local Square
do
	Square = setmetatable({}, {
		__tostring = function()
			return "Square"
		end,
	})
	Square.__index = Square
	function Square.new(...)
		local self = setmetatable({}, Square)
		return self:constructor(...) or self
	end
	function Square:constructor()
		self.sides = 4
	end
end
local makeSquare = function()
	return Square.new()
end
return {
	Square = Square,
	makeSquare = makeSquare,
}
