-- Compiled with roblox-ts v3.0.0
local TS = _G[script]
local Roact = TS.import(script, TS.getModule(script, "@rbxts", "roact").src.ts)
local function Widget(props)
	return Roact.jsx("frame", {
		Key = "x",
		Size = UDim2.new(0, props.model.sides, 0, 1),
	})
end
local element = Roact.jsx("textlabel", {
	Text = "y",
})
return {
	default = Widget,
	element = element,
}
