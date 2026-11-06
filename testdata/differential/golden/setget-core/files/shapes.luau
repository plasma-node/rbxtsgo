-- Compiled with roblox-ts v3.0.0
local Holder
do
	Holder = setmetatable({}, {
		__tostring = function()
			return "Holder"
		end,
	})
	Holder.__index = Holder
	function Holder.new(...)
		local self = setmetatable({}, Holder)
		return self:constructor(...) or self
	end
	function Holder:constructor()
		self.inner = 1
		self.field = 1
	end
	function Holder:__getv()
		return self.inner
	end
	function Holder:__setv(val)
		self.inner = val
	end
	function Holder:__getval()
		return self
	end
	function Holder:__setval(value)
		print(value)
	end
end
local h = Holder.new()
local maybe = h
-- optional chaining
local _self = maybe
local _result = _self
if _result ~= nil then
	_result = _result.__getv
	if _result ~= nil then
		_result = _result(_self)
	end
end
print(_result)
local _self_1 = maybe
local _result_1 = _self_1
if _result_1 ~= nil then
	_result_1 = _result_1.__getval
	if _result_1 ~= nil then
		local _self_2 = _result_1(_self_1)
		_result_1 = _self_2
		if _result_1 ~= nil then
			_result_1 = _result_1.__getv
			if _result_1 ~= nil then
				_result_1 = _result_1(_self_2)
			end
		end
	end
end
print(_result_1)
local _self_2 = h:__getval()
local _result_2 = _self_2
if _result_2 ~= nil then
	_result_2 = _result_2.__getv
	if _result_2 ~= nil then
		_result_2 = _result_2(_self_2)
	end
end
print(_result_2)
local _self_3 = h.maybe
local _result_3 = _self_3
if _result_3 ~= nil then
	_result_3 = _result_3.__getval
	if _result_3 ~= nil then
		_result_3 = _result_3(_self_3)
		if _result_3 ~= nil then
			_result_3 = _result_3.field
		end
	end
end
print(_result_3)
local _result_4 = h.maybe
if _result_4 ~= nil then
	_result_4 = _result_4.field
end
print(_result_4)
-- non-null, parens and casts
print((h:__getval()):__getv());
(h:__getval()):__setv(2);
(h:__getval()):__setv((h:__getval()):__getv() + 2)
h:__setv(3)
h:__setv(4)
h:__setv(5)
h:__setv(6)
print(h:__getv())
print(h:__getv())
-- not a set when it's only under the assignment's right side
local arr = { 1, 2 }
h:__setv(1)
local plain = 0
plain = h.field
print(plain)
local obj = {
	key = 1,
}
obj.key = h.field
obj.key += h.field
print(obj)
-- the first property access anywhere under the expression decides
h:__setv(h:__getv())
h:__setv(h:__getv() + h:__getv())
print(plain)
h:__setv(h:__getv())
local y = (h:__setv(h:__getv()))
print(y)
local z = if h:__getv() > 1 then h:__getv() else 0
h:__setv(if h:__getv() > 1 then h.field else 0)
print(z)
print(h:__getv() + h:__getv(), h:__getv() == 1, h:__getv() ~= h.field)
h:__setv(h:__getv() + 1)
h:__setv(h.field)
h:__setval(h)
h:__setval(h:__getval())
h.field = h:__getv()
local nested = (h:__setv(1))
print(nested)
-- calls and declarations
local function take(a, b)
	local _exp = a
	local _condition = b
	if _condition == nil then
		_condition = 0
	end
	return _exp + _condition
end
print(take(h:__getv()), take(h:__getv(), h:__getv()))
local fn = function()
	return h:__getv()
end
print(fn())
local typed = h:__getv()
print(typed)
local first = h:__getv()
local _binding = {
	key = h:__getv(),
}
local renamed = _binding.key
print(first, renamed)
local _array = {}
local _length = #_array
local _array_1 = { h:__getv() }
table.move(_array_1, 1, #_array_1, _length + 1, _array)
local spreadIt = _array
print(spreadIt)
print(Holder.new():__getv())
Holder.new():__setv(1)
print(h.field)
return nil
