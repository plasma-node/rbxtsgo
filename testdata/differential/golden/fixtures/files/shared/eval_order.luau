-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
-- Evaluation order, prereqs and temp names.
local function side(n)
	return n
end
local arr = { 1, 2, 3 }
local map = {}
local obj = {
	a = 1,
	b = {
		c = 2,
	},
}
local i = 0
local _exp = arr
local _original = i
i += 1
local _exp_1 = _exp[_original + 1]
local _exp_2 = arr
local _original_1 = i
i += 1
local r1 = _exp_1 + _exp_2[_original_1 + 1]
local _array = { side(1) }
local _length = #_array
local _arrLength = #arr
table.move(arr, 1, _arrLength, _length + 1, _array)
_length += _arrLength
_array[_length + 1] = side(2)
local _original_2 = i
i += 1
_array[_length + 2] = _original_2
table.move(arr, 1, #arr, _length + 3, _array)
local r2 = _array
local _object = {}
local _left = "x"
local _original_3 = i
i += 1
_object[_left] = _original_3
local _original_4 = i
i += 1
local _left_1 = `k{_original_4}`
_object[_left_1] = side(i)
for _k, _v in obj do
	_object[_k] = _v
end
_object.y = side(3)
local r3 = _object
local _original_5 = i
i += 1
local _result
local _value = side(_original_5)
if _value ~= 0 and _value == _value and _value then
	local _result_1
	if i > 2 then
		local _original_6 = i
		i += 1
		_result_1 = side(_original_6)
	else
		local _original_6 = i
		i -= 1
		_result_1 = side(_original_6)
	end
	_result = _result_1
else
	_result = side(0)
end
local r4 = _result
local _index = side(1) + 1
arr[_index] += side(2)
arr[side(0) + 1] = arr[_index]
i = 5
obj.a = i
obj.b.c = obj.a
i = 6
local _exp_3 = i
i += 1
local r5 = _exp_3 + i
local _arg1 = side(1)
map.a = _arg1
local _condition = map.a
if _condition == nil then
	_condition = side(2)
end
local r6 = _condition
local _condition_1 = obj.a
if _condition_1 ~= 0 and _condition_1 == _condition_1 and _condition_1 then
	_condition_1 = obj.b and side(1)
end
local r7 = _condition_1
local _original_6 = i
i += 1
local _condition_2 = _original_6
if not (_condition_2 ~= 0 and _condition_2 == _condition_2 and _condition_2) then
	local _original_7 = i
	i -= 1
	_condition_2 = _original_7
	if not (_condition_2 ~= 0 and _condition_2 == _condition_2 and _condition_2) then
		_condition_2 = side(0)
	end
end
local r8 = _condition_2
local maybe
local _r9 = maybe
if _r9 ~= nil then
	_r9 = _r9.f
	if _r9 ~= nil then
		_r9 = _r9(side(1))
	end
end
local r9 = _r9
local _result_1 = maybe
if _result_1 ~= nil then
	_result_1 = _result_1.g.h
end
local _condition_3 = _result_1
if _condition_3 == nil then
	_condition_3 = 2
end
local r10 = _condition_3
local _binding = arr
local p = _binding[1]
local q = _binding[3]
if q == nil then
	q = side(1)
end
local _binding_1 = obj
local renamed = _binding_1.a
local _binding_2 = _binding_1.b
local deep = _binding_2.c
if deep == nil then
	deep = 3
end
do
	local _k = 0
	local m = side(1)
	local _shouldIncrement = false
	while true do
		local k = _k
		if _shouldIncrement then
			k += 1
		else
			_shouldIncrement = true
		end
		if not (k < m) then
			break
		end
		local fn = function()
			return k
		end
		if fn() > 0 then
			_k = k
			continue
		end
		if k > 3 then
			break
		end
		_k = k
	end
end
for key, value in map do
	print(key, value)
end
for _, v in arr do
	print(v)
end
for c in string.gmatch("héllo", utf8.charpattern) do
	print(c)
end
local function gen()
	return TS.generator(function()
		coroutine.yield(1)
		local got = coroutine.yield(2)
		print(got)
		return 3
	end)
end
for _result_2 in gen().next do
	if _result_2.done then
		break
	end
	local g = _result_2.value
	print(g)
end
local _exp_4 = side(1)
repeat
	local _fallthrough = false
	if _exp_4 == (side(1)) then
		_fallthrough = true
	end
	if _fallthrough or _exp_4 == 2 then
		print("a")
		_fallthrough = true
	end
	if _fallthrough or _exp_4 == 3 then
		print("b")
		break
	end
	print("c")
until true
TS.try(function()
	print(side(1))
	error("x")
end, function(e)
	print(e)
end, function()
	print("done")
end)
local function usesTry()
	local _exitType, _returns = TS.try(function()
		return TS.TRY_RETURN, { side(1) }
	end, function()
		return TS.TRY_RETURN, { side(2) }
	end)
	if _exitType then
		return unpack(_returns)
	end
end
local hoisted
print(hoisted())
local later
function hoisted()
	return later
end
later = 1
local label = `{side(1)}-{"x"}-{i}`
local typeChecks = typeof(label) == "string" and type(label) == "string"
return {
	r1 = r1,
	r2 = r2,
	r3 = r3,
	r4 = r4,
	r5 = r5,
	r6 = r6,
	r7 = r7,
	r8 = r8,
	r9 = r9,
	r10 = r10,
	p = p,
	q = q,
	renamed = renamed,
	deep = deep,
	usesTry = usesTry,
	label = label,
	typeChecks = typeChecks,
}
