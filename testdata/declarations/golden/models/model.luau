-- Compiled with roblox-ts v3.0.0
local TS = _G[script]
--/ <reference types="@rbxts/types" />
local makeSquare = TS.import(script, script.Parent, "shapes").makeSquare
local help = TS.import(script, script.Parent.Parent, "util", "helper").helper
local unrelated = TS.import(script, script.Parent, "deep", "other")
-- The return type here names a type from another module without importing it, so the declaration has import("...").
local function inferred()
	return makeSquare()
end
local viaHelper = help(1)
local fromRequire = unrelated.other
local function usesInstance()
	return Instance.new("Part")
end
local function usesFolder(parent, name)
	return {
		folder = parent:FindFirstChild("x"),
		name = name,
	}
end
local Model
do
	Model = setmetatable({}, {
		__tostring = function()
			return "Model"
		end,
	})
	Model.__index = Model
	function Model.new(...)
		local self = setmetatable({}, Model)
		return self:constructor(...) or self
	end
	function Model:constructor()
		self.sides = 3
	end
	function Model:shape()
		return self
	end
end
return {
	inferred = inferred,
	usesInstance = usesInstance,
	usesFolder = usesFolder,
	viaHelper = viaHelper,
	fromRequire = fromRequire,
	Model = Model,
}
