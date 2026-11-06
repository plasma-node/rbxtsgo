-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local Roact = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "roact", "src", "ts")
local Unicode
do
	Unicode = setmetatable({}, {
		__tostring = function()
			return "Unicode"
		end,
	})
	Unicode.__index = Unicode
	function Unicode.new(...)
		local self = setmetatable({}, Unicode)
		return self:constructor(...) or self
	end
	function Unicode:constructor()
	end
	function Unicode:__getword()
		return "日本語😀"
	end
	function Unicode:__setword(val)
		print(val)
	end
end
local u = Unicode.new()
u:__setword("😀")
print(u:__getword(), u:__getword())
local Sized
do
	Sized = setmetatable({}, {
		__tostring = function()
			return "Sized"
		end,
	})
	Sized.__index = Sized
	function Sized.new(...)
		local self = setmetatable({}, Sized)
		return self:constructor(...) or self
	end
	function Sized:constructor()
	end
	function Sized:__getwidth()
		return 1
	end
end
local sized = Sized.new()
-- JSX goes through the printer as well
local element = (Roact.jsx("frame", {
	Size = UDim2.new(sized:__getwidth(), 0, 0, 0),
	Key = "k",
}, Roact.jsx("textlabel", {
	Text = "a" .. u:__getword(),
})))
print(element)
return nil
