-- Compiled with roblox-ts v3.0.0
local arr = { 3, 1, 2 }
local readonlyArr = arr
local set = {
	[1] = true,
}
local map = {
	a = 1,
}
local s = "héllo 日本語"
local _exp = #arr + #readonlyArr
-- ▼ ReadonlySet.size ▼
local _size = 0
for _ in set do
	_size += 1
end
-- ▲ ReadonlySet.size ▲
local _exp_1 = _exp + _size
-- ▼ ReadonlyMap.size ▼
local _size_1 = 0
for _ in map do
	_size_1 += 1
end
-- ▲ ReadonlyMap.size ▲
local a1 = _exp_1 + _size_1 + #s
-- ▼ Array.push ▼
table.insert(arr, 4)
table.insert(arr, 5)
-- ▲ Array.push ▲
table.insert(arr, 1, 0)
-- ▼ Array.pop ▼
local _length = #arr
local _result = arr[_length]
arr[_length] = nil
-- ▲ Array.pop ▲
local a2 = _result
local a3 = table.remove(arr, 1)
-- ▼ ReadonlyArray.map ▼
local _newValue = table.create(#arr)
local _callback = function(v)
	return v * 2
end
for _k, _v in arr do
	_newValue[_k] = _callback(_v, _k - 1, arr)
end
-- ▲ ReadonlyArray.map ▲
-- ▼ ReadonlyArray.filter ▼
local _newValue_1 = {}
local _callback_1 = function(v)
	return v > 1
end
local _length_1 = 0
for _k, _v in _newValue do
	if _callback_1(_v, _k - 1, _newValue) == true then
		_length_1 += 1
		_newValue_1[_length_1] = _v
	end
end
-- ▲ ReadonlyArray.filter ▲
local a4 = _newValue_1
-- ▼ ReadonlyArray.reduce ▼
local _result_1 = 0
local _callback_2 = function(acc, v)
	return acc + v
end
for _i = 1, #arr do
	_result_1 = _callback_2(_result_1, arr[_i], _i - 1, arr)
end
-- ▲ ReadonlyArray.reduce ▲
local a5 = _result_1
local a6 = table.find(arr, 1) ~= nil and (table.find(arr, 2) or 0) - 1 > 0
local a7 = table.concat(arr, ", ")
table.sort(arr, function(x, y)
	return x < y
end)
-- ▼ ReadonlyArray.forEach ▼
local _callback_3 = function(v)
	return print(v)
end
for _k, _v in arr do
	_callback_3(_v, _k - 1, arr)
end
-- ▲ ReadonlyArray.forEach ▲
-- ▼ ReadonlyArray.find ▼
local _callback_4 = function(v)
	return v > 1
end
local _result_2
for _i, _v in arr do
	if _callback_4(_v, _i - 1, arr) == true then
		_result_2 = _v
		break
	end
end
-- ▲ ReadonlyArray.find ▲
local a8 = _result_2
-- ▼ ReadonlyArray.findIndex ▼
local _callback_5 = function(v)
	return v > 1
end
local _result_3 = -1
for _i, _v in arr do
	if _callback_5(_v, _i - 1, arr) == true then
		_result_3 = _i - 1
		break
	end
end
-- ▲ ReadonlyArray.findIndex ▲
local a9 = _result_3
-- ▼ ReadonlyArray.every ▼
local _result_4 = true
local _callback_6 = function(v)
	return v > 0
end
for _k, _v in arr do
	if not _callback_6(_v, _k - 1, arr) then
		_result_4 = false
		break
	end
end
-- ▲ ReadonlyArray.every ▲
local _condition = _result_4
if not _condition then
	-- ▼ ReadonlyArray.some ▼
	local _result_5 = false
	local _callback_7 = function(v)
		return v > 0
	end
	for _k, _v in arr do
		if _callback_7(_v, _k - 1, arr) then
			_result_5 = true
			break
		end
	end
	-- ▲ ReadonlyArray.some ▲
	_condition = _result_5
end
local a10 = _condition
table.insert(arr, 2, 2)
table.remove(arr, 1)
-- ▼ Array.unorderedRemove ▼
local _length_2 = #arr
local _value = arr[1]
if _value ~= nil then
	arr[1] = arr[_length_2]
	arr[_length_2] = nil
end
-- ▲ Array.unorderedRemove ▲
table.clear(arr)
set[1] = true
set[2] = true
set[1] = nil
-- ▼ ReadonlySet.forEach ▼
local _callback_7 = function(v)
	return print(v)
end
for _v in set do
	_callback_7(_v, _v, set)
end
-- ▲ ReadonlySet.forEach ▲
local a11 = set[2] ~= nil
map.x = 1
map.y = 2
map.x = nil
-- ▼ ReadonlyMap.forEach ▼
local _callback_8 = function(v, k)
	return print(k, v)
end
for _k, _v in map do
	_callback_8(_v, _k, map)
end
-- ▲ ReadonlyMap.forEach ▲
local a12 = if map.y ~= nil then map.y else 0
local a13 = string.sub(s, 1, 2) .. string.upper(s) .. string.lower(s) .. string.rep(s, 2) .. string.reverse(s)
local a14 = { string.byte(s, 1) }
local a15 = { string.find(s, "l") }
local a16 = string.split(s, " ")
local a17 = { string.gsub(s, "l", "L") }
local a18 = table.create(3)
local a19 = {
	[1] = true,
	[2] = true,
}
local a20 = {
	k = 1,
}
local _vector3 = Vector3.new(1, 2, 3)
local _vector3_1 = Vector3.new(1, 1, 1)
local a21 = (_vector3 + _vector3_1) * 2
local a22 = 1
for n = 1, 3 do
	print(n)
end
local function tup()
	return 1, "a"
end
local t1, t2 = tup()
return {
	a1 = a1,
	a2 = a2,
	a3 = a3,
	a4 = a4,
	a5 = a5,
	a6 = a6,
	a7 = a7,
	a8 = a8,
	a9 = a9,
	a10 = a10,
	a11 = a11,
	a12 = a12,
	a13 = a13,
	a14 = a14,
	a15 = a15,
	a16 = a16,
	a17 = a17,
	a18 = a18,
	a19 = a19,
	a20 = a20,
	a21 = a21,
	a22 = a22,
	t1 = t1,
	t2 = t2,
}
