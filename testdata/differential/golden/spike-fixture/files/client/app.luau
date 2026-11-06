-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local _react = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "react")
local React = _react
local useState = _react.useState
local _react_roblox = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "react-roblox")
local createPortal = _react_roblox.createPortal
local createRoot = _react_roblox.createRoot
local Players = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "services").Players
local function Label(_param)
	local text = _param.text
	local size = _param.size
	return React.createElement("textlabel", {
		Text = text,
		Size = size or UDim2.new(1, 0, 0, 20),
		BackgroundTransparency = 1,
	})
end
local function App()
	local count, setCount = useState(0)
	return React.createElement(React.Fragment, nil, React.createElement(Label, {
		text = `count {count}`,
	}), React.createElement("textbutton", {
		Text = "more",
		Event = {
			Activated = function()
				return setCount(count + 1)
			end,
		},
	}))
end
local playerGui = Players.LocalPlayer:WaitForChild("PlayerGui")
local root = createRoot(Instance.new("Folder"))
root:render(createPortal(React.createElement(App), playerGui))
return nil
