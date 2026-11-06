-- Compiled with roblox-ts v3.0.0
local function Label(_param)
	local text = _param.text
	local order = _param.order
	if order == nil then
		order = 0
	end
	return React.createElement("textlabel", {
		Text = text,
		LayoutOrder = order,
	})
end
local function Counter(props)
	local count, setCount = React:useState(props.start)
	local visible, setVisible = React:useState(true)
	return React.createElement("frame", {
		Size = UDim2.new(1, 0, 1, 0),
	}, React.createElement(Label, {
		key = "count",
		text = `count: {count}`,
		order = 1,
	}), visible and React.createElement(Label, {
		key = "visible",
		text = "visible",
	}), if not visible then React.createElement(Label, {
		key = "hidden",
		text = "hidden",
	}) else nil, React.createElement("textbutton", {
		Text = "inc",
		Event = {
			Activated = function()
				return setCount(count + 1)
			end,
		},
	}), React.createElement("textbutton", {
		Text = "toggle",
		Event = {
			Activated = function()
				return setVisible(function(v)
					return not v
				end)
			end,
		},
	}))
end
local function List(props)
	local _exp = props.items
	-- ▼ ReadonlyArray.map ▼
	local _newValue = table.create(#_exp)
	local _callback = function(item, index)
		return React.createElement(Label, {
			key = `item-{index}`,
			text = item,
			order = index,
		})
	end
	for _k, _v in _exp do
		_newValue[_k] = _callback(_v, _k - 1, _exp)
	end
	-- ▲ ReadonlyArray.map ▲
	return React.createElement(React.Fragment, nil, _newValue)
end
local function Spread(props)
	local base = {
		text = "base",
	}
	local _attributes = table.clone(base)
	setmetatable(_attributes, nil)
	local _exp = React.createElement(Label, _attributes)
	local _attributes_1 = table.clone(base)
	setmetatable(_attributes_1, nil)
	for _k, _v in props.extra do
		_attributes_1[_k] = _v
	end
	_attributes_1.order = 5
	local _exp_1 = React.createElement(Label, _attributes_1)
	local _attributes_2 = {
		text = "late",
	}
	for _k, _v in props.extra do
		_attributes_2[_k] = _v
	end
	local _exp_2 = React.createElement(Label, _attributes_2)
	local _attributes_3 = table.clone({
		Size = UDim2.new(0, 1, 0, 1),
	})
	setmetatable(_attributes_3, nil)
	return React.createElement(React.Fragment, nil, _exp, _exp_1, _exp_2, React.createElement("frame", _attributes_3))
end
local function Wrapper(props)
	return React.createElement("frame", nil, props.children)
end
local tree = (React.createElement(Wrapper, nil, React.createElement(Counter, {
	start = 3,
}), React.createElement(List, {
	items = { "a", "b" },
}), React.createElement(Spread, {
	extra = {
		text = "x",
	},
}), React.createElement(Label, {
	text = "plain",
}, "text child"), { React.createElement(Label, {
	key = "a",
	text = "a",
}), React.createElement(Label, {
	key = "b",
	text = "b",
}) }))
return {
	tree = tree,
}
