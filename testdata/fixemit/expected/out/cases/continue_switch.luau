-- Compiled with roblox-ts v3.0.0
-- Upstream PR #3125's runtime tests, in tests/src/tests/switch.spec.ts, printing what they compare. Issue #2899.
local function forOfLoop()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			if value == 2 then
				table.insert(visited, "skipped")
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
	print("for-of", table.concat(visited, ","))
end
local function whileLoop()
	local visited = {}
	local count = 0
	while count < 3 do
		count += 1
		local _continue = false
		repeat
			if count == 2 then
				_continue = true
				break
			end
		until true
		if _continue then
			continue
		end
		local _count = count
		table.insert(visited, _count)
	end
	print("while", table.concat(visited, ","))
end
local function nestedSwitch()
	local visited = {}
	for _, value in { 1, 2 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 2 then
				local _continue_1 = false
				repeat
					if value == 2 then
						table.insert(visited, "inner")
						_continue_1 = true
						break
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
	print("nested switch", table.concat(visited, ","))
end
local function loopInsideCase()
	local visited = {}
	for _, outer in { 1, 2 } do
		repeat
			if outer == 1 then
				for _1, inner in { 1, 2, 3 } do
					if inner == 2 then
						continue
					end
					local _arg0 = `inner {inner}`
					table.insert(visited, _arg0)
				end
				break
			end
		until true
		local _arg0 = `outer {outer}`
		table.insert(visited, _arg0)
	end
	print("loop inside case", table.concat(visited, ","))
end
local function writeBack()
	local visited = {}
	-- writing the loop variable in the body forces the unoptimized loop form, which copies it back every iteration
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
			if not (i < 6) then
				break
			end
			-- a dropped write back turns the continue below into an infinite loop
			guard += 1
			if guard > 20 then
				break
			end
			local _continue = false
			repeat
				if i == 1 then
					_continue = true
					break
				end
				if i == 3 then
					i += 1
					break
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
	print("write back", table.concat(visited, ","), guard)
end
local function doWhile()
	local visited = {}
	local count = 0
	repeat
		do
			count += 1
			local _continue = false
			repeat
				if count == 2 then
					_continue = true
					break
				end
			until true
			if _continue then
				continue
			end
			local _count = count
			table.insert(visited, _count)
		end
	until not (count < 3)
	print("do-while", table.concat(visited, ","))
end
local function fallthroughAndBreak()
	local visited = {}
	for _, value in { 1, 2, 3, 4 } do
		local _continue = false
		repeat
			local _fallthrough = false
			if value == 1 then
				_fallthrough = true
			end
			if _fallthrough or value == 2 then
				if value == 2 then
					_continue = true
					break
				end
				table.insert(visited, "one or two")
				break
			end
			if value == 3 then
				break
			end
			table.insert(visited, "default")
		until true
		if _continue then
			continue
		end
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("fallthrough", table.concat(visited, ","))
end
local function multipleContinues()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			if value == 1 then
				_continue = true
				break
			end
			if value == 2 then
				_continue = true
				break
			end
		until true
		if _continue then
			continue
		end
		table.insert(visited, value)
	end
	print("multiple", table.concat(visited, ","))
end
local function defaultClause()
	local visited = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			if value == 1 then
				break
			end
			_continue = true
			break
		until true
		if _continue then
			continue
		end
		table.insert(visited, value)
	end
	print("default clause", table.concat(visited, ","))
end
local function noContinue()
	local visited = {}
	for _, value in { 1, 2 } do
		repeat
			if value == 1 then
				table.insert(visited, "a")
				break
			end
			table.insert(visited, "b")
		until true
		local _arg0 = `after {value}`
		table.insert(visited, _arg0)
	end
	print("no continue", table.concat(visited, ","))
end
local function closureInLoop()
	local callbacks = {}
	for _, value in { 1, 2, 3 } do
		local _continue = false
		repeat
			if value == 2 then
				_continue = true
				break
			end
		until true
		if _continue then
			continue
		end
		table.insert(callbacks, function()
			return value
		end)
	end
	local values = {}
	for _, callback in callbacks do
		local _arg0 = callback()
		table.insert(values, _arg0)
	end
	print("closure", table.concat(values, ","))
end
forOfLoop()
whileLoop()
nestedSwitch()
loopInsideCase()
writeBack()
doWhile()
fallthroughAndBreak()
multipleContinues()
defaultClause()
noContinue()
closureInLoop()
return nil
