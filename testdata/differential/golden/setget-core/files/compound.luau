-- Compiled with roblox-ts v3.0.0
local Counter
do
	Counter = setmetatable({}, {
		__tostring = function()
			return "Counter"
		end,
	})
	Counter.__index = Counter
	function Counter.new(...)
		local self = setmetatable({}, Counter)
		return self:constructor(...) or self
	end
	function Counter:constructor()
		self.inner = 1
	end
	function Counter:__getn()
		return self.inner
	end
	function Counter:__setn(val)
		self.inner = val
	end
	function Counter:__getflag()
		return true
	end
	function Counter:__setflag(val)
		print(val)
	end
	function Counter:__getself()
		return self
	end
	function Counter:__setself(val)
		print(val)
	end
end
local c = Counter.new()
local k = 2
c:__setn(c:__getn() + 1)
c:__setn(c:__getn() - 1)
c:__setn(c:__getn() * 2)
c:__setn(c:__getn() / 2)
c:__setn(c:__getn() % 2)
c:__setn(c:__getn() ^ 2)
c:__setn(bit32.lshift(c:__getn(), 1))
c:__setn(bit32.arshift(c:__getn(), 1))
c:__setn(bit32.rshift(c:__getn(), 1))
c:__setn(bit32.band(c:__getn(), 1))
c:__setn(bit32.bor(c:__getn(), 1))
c:__setn(bit32.bxor(c:__getn(), 1))
c:__setflag(c:__getflag() or true)
c:__setflag(c:__getflag() and false)
local _condition = c:__getflag()
if _condition == nil then
	_condition = true
end
c:__setflag(_condition)
c:__setn(c:__getn() + (k + 1))
c:__setn(c:__getn() - (k - 1))
c:__setn(c:__getn() - (k + 1))
c:__setn(c:__getn() * (k + 1))
c:__setn(c:__getn() * k * 2)
c:__setn(c:__getn() / (k / 2))
c:__setn(c:__getn() ^ (k ^ 2))
c:__setn(c:__getn() ^ (k * 2))
c:__setn(bit32.band(bit32.band(c:__getn(), k), 1))
c:__setn(bit32.band(c:__getn(), (bit32.bor(k, 1))))
c:__setn(bit32.bor(bit32.bor(c:__getn(), k), 1))
c:__setn(bit32.bxor(bit32.bxor(c:__getn(), k), 1))
c:__setn(bit32.lshift(c:__getn(), (bit32.lshift(k, 1))))
c:__setn(c:__getn() + (k + 1))
c:__setn(c:__getn() + (if k > 1 then 1 else 2))
c:__setn(c:__getn() + c:__getn())
c:__setn(c:__getn() * (c:__getn() + c:__getn()))
c:__setflag(c:__getflag() or k > 1 and k < 3)
c:__setflag(c:__getflag() and (k > 1 or k < 3))
local _condition_1 = c:__getflag()
if _condition_1 == nil then
	_condition_1 = k > 1
end
c:__setflag(if _condition_1 then true else nil);
(c:__getself()):__setn((c:__getself()):__getn() + 1);
((c:__getself()):__getself()):__setn(((c:__getself()):__getself()):__getn() * 2)
c:__setn(c:__getn() + 1)
c:__setn(c:__getn() - 1)
c:__setn(c:__getn())
c:__setn(c:__getn() + 1)
local before = c:__setn(c:__getn() + 1)
print(before, k)
do
	local _i = 0
	local _shouldIncrement = false
	while true do
		local i = _i
		if _shouldIncrement then
			c:__setn(c:__getn() + 1)
		else
			_shouldIncrement = true
		end
		if not (i < 2) then
			break
		end
		i += 1
		_i = i
	end
end
local function getCounter()
	print("called")
	return c
end
getCounter():__setn(getCounter():__getn() + 1)
getCounter():__setn(getCounter():__getn() + 1)
local arr = { c }
arr[1]:__setn(arr[1]:__getn() + 1)
arr[1]:__setn(arr[1]:__getn() + 1)
arr[k + 1]:__setn(3)
return nil
