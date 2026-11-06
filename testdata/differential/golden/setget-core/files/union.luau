-- Compiled with roblox-ts v3.0.0
local Left
do
	Left = setmetatable({}, {
		__tostring = function()
			return "Left"
		end,
	})
	Left.__index = Left
	function Left.new(...)
		local self = setmetatable({}, Left)
		return self:constructor(...) or self
	end
	function Left:constructor()
	end
	function Left:__getshared()
		return 1
	end
	function Left:__setshared(val)
		print(val)
	end
end
local Right
do
	Right = setmetatable({}, {
		__tostring = function()
			return "Right"
		end,
	})
	Right.__index = Right
	function Right.new(...)
		local self = setmetatable({}, Right)
		return self:constructor(...) or self
	end
	function Right:constructor()
	end
	function Right:__getshared()
		return 2
	end
	function Right:__setshared(val)
		print(val)
	end
end
local either = (if math.random() > 0.5 then Left.new() else Right.new())
print(either:__getshared())
either:__setshared(1)
either:__setshared(either:__getshared() + 1)
print(both:__getshared(), both.more)
local function generic(value)
	value:__setshared(1)
	return value:__getshared()
end
print(generic(Left.new()))
return nil
