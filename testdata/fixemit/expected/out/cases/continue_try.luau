-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
-- A continue that goes through a try as well as a switch. Upstream PR #3125 leaves these with the old bug: its snapshot
-- "avoids continue flags for try statements inside a case" still has a bare continue inside the repeat.
local function tryInsideCase()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 2 then
				local _exitType, _returns = TS.try(function()
					table.insert(visited, "try")
					return TS.TRY_CONTINUE
				end, nil, function()
					table.insert(visited, "finally")
				end)
				if _exitType then
					_continue = true
					break
				end
			end
		until true
		if _continue then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("try inside case", table.concat(visited, ","))
end
local function catchInsideCase()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 2 then
				local _exitType, _returns = TS.try(function()
					error("x")
				end, function(e)
					table.insert(visited, "catch")
					return TS.TRY_CONTINUE
				end)
				if _exitType then
					_continue = true
					break
				end
			end
		until true
		if _continue then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("catch inside case", table.concat(visited, ","))
end
local function switchInsideTry()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _exitType, _returns = TS.try(function()
			local _continue = false
			repeat
				if value == 2 then
					table.insert(visited, "case")
					_continue = true
					break
				end
			until true
			if _continue then
				return TS.TRY_CONTINUE
			end
			local _arg0 = `in try {value}`
			table.insert(visited, _arg0)
		end, nil, function()
			local _arg0 = `finally {value}`
			table.insert(visited, _arg0)
		end)
		if _exitType then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("switch inside try", table.concat(visited, ","))
end
local function tryInSwitchInTry()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _exitType, _returns = TS.try(function()
			local _continue = false
			repeat
				local _fallthrough = false
				if value == 2 then
					local _exitType_1, _returns_1 = TS.try(function()
						return TS.TRY_CONTINUE
					end, nil, function()
						table.insert(visited, "inner finally")
					end)
					if _exitType_1 then
						_continue = true
						break
					end
				end
			until true
			if _continue then
				return TS.TRY_CONTINUE
			end
			local _arg0 = `in try {value}`
			table.insert(visited, _arg0)
		end, nil, function()
			local _arg0 = `outer finally {value}`
			table.insert(visited, _arg0)
		end)
		if _exitType then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("try in switch in try", table.concat(visited, ","))
end
local function tryInNestedSwitch()
	local visited = {}
	for _, value in { 1, 2 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 2 then
				local _continue_1 = false
				repeat
					local _fallthrough_1 = false
					if value == 2 then
						local _exitType, _returns = TS.try(function()
							return TS.TRY_CONTINUE
						end, function(e)
							table.insert(visited, "not reached")
						end)
						if _exitType then
							_continue_1 = true
							break
						end
					end
				until true
				if _continue_1 then
					_continue = true
					break
				end
				table.insert(visited, "outer")
			end
		until true
		if _continue then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("try in nested switch", table.concat(visited, ","))
end
local function writeBackThroughTry()
	local visited = {}
	local guard = 0
	do
		local _i = 0
		local _shouldIncrement = false
		while true do
			local i = _i
			if _shouldIncrement then
				i += 1
			else
				_shouldIncrement = true
			end
			if not (i < 5) then
				break
			end
			guard += 1
			if guard > 20 then
				break
			end
			local _continue = false
			repeat
				local _fallthrough = false
				if i == 1 then
					local _exitType, _returns = TS.try(function()
						i += 1
						return TS.TRY_CONTINUE
					end, nil, function()
						local _arg0 = -i
						table.insert(visited, _arg0)
					end)
					if _exitType then
						_continue = true
						break
					end
				end
			until true
			if _continue then
				_i = i
				continue
			end
			local _i_1 = i
			table.insert(visited, _i_1)
			_i = i
		end
	end
	print("write back through try", table.concat(visited, ","), guard)
end
local function whileThroughTry()
	local visited = {}
	local count = 0
	while count < 3 do
		count += 1
		local _continue = false
		repeat
			local _fallthrough = false
			if count == 2 then
				local _exitType, _returns = TS.try(function()
					return TS.TRY_CONTINUE
				end, function(e) end)
				if _exitType then
					_continue = true
					break
				end
			end
		until true
		if _continue then
			continue
		end
		local _count = count
		table.insert(visited, _count)
	end
	print("while through try", table.concat(visited, ","))
end
-- These work without the fix, and have to keep working.
local function breakThroughTry()
	local visited = {}
	for _, value in { 1, 2 } do
		repeat
			local _fallthrough = false
			if value == 1 then
				local _exitType, _returns = TS.try(function()
					return TS.TRY_BREAK
				end, nil, function()
					table.insert(visited, "finally")
				end)
				if _exitType then
					break
				end
			end
		until true
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("break through try", table.concat(visited, ","))
end
local function returnThroughTry()
	for _, value in { 1, 2 } do
		repeat
			local _fallthrough = false
			if value == 2 then
				local _exitType, _returns = TS.try(function()
					return TS.TRY_RETURN, { value * 10 }
				end, nil, function()
					print("return finally")
				end)
				if _exitType then
					return unpack(_returns)
				end
			end
		until true
	end
	return 0
end
local function breakAndContinueInTry()
	local visited = {}
	for _, value in { 1, 2, 3, 4 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 2 then
				_fallthrough = true
			end
			if _fallthrough or value == 3 then
				local _exitType, _returns = TS.try(function()
					if value == 2 then
						return TS.TRY_CONTINUE
					end
					return TS.TRY_BREAK
				end, nil, function()
					local _arg0 = `finally {value}`
					table.insert(visited, _arg0)
				end)
				if _exitType == TS.TRY_BREAK then
					break
				elseif _exitType then
					_continue = true
					break
				end
			end
		until true
		if _continue then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("break and continue", table.concat(visited, ","))
end
local function tryAroundLoop()
	local visited = {}
	TS.try(function()
		for _, value in { 1, 2 } do
			local _continue = false
			repeat
				if value == 1 then
					_continue = true
					break
				end
			until true
			if _continue then
				continue
			end
			local _arg0 = `after {value}`
			table.insert(visited, _arg0)
		end
	end, nil, function()
		table.insert(visited, "finally")
	end)
	print("try around loop", table.concat(visited, ","))
end
tryInsideCase()
catchInsideCase()
switchInsideTry()
tryInSwitchInTry()
tryInNestedSwitch()
writeBackThroughTry()
whileThroughTry()
breakThroughTry()
print("return through try", returnThroughTry())
breakAndContinueInTry()
tryAroundLoop()
return nil
