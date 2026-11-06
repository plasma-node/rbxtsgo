-- Compiled with roblox-ts v3.0.0
local Basic
do
	Basic = setmetatable({}, {
		__tostring = function()
			return "Basic"
		end,
	})
	Basic.__index = Basic
	function Basic.new(...)
		local self = setmetatable({}, Basic)
		return self:constructor(...) or self
	end
	function Basic:constructor()
		self.value = 1
		self.other = "text"
	end
	function Basic:__getboth()
		return self.value
	end
	function Basic:__setboth(val)
		self.value = val
	end
	function Basic:__getonlyGet()
		return self.other
	end
	function Basic:__setonlySet(val)
		self.other = val
	end
	function Basic:__gettotal()
		return Basic.count
	end
	function Basic:__settotal(val)
		Basic.count = val
	end
	function Basic:viaThis()
		self:__setboth(self:__getboth() + 1)
		self:__setonlySet(self:__getonlyGet())
		return self:__getboth()
	end
	Basic.count = 0
end
local basic = Basic.new()
basic:__setboth(5)
print(basic:__getboth())
print(basic:__getonlyGet())
basic:__setonlySet("val")
Basic:__settotal(3)
print(Basic:__gettotal())
print(basic:viaThis())
print(Basic.new():__getboth())
local fromCall = function()
	return basic
end
print(fromCall():__getboth())
fromCall():__setboth(2)
local _arg0 = basic:__getboth()
print(typeof(_arg0))
print(`{basic:__getboth()} and {basic:__getonlyGet()}`)
local list = { basic:__getboth(), basic:__getboth() }
local record = {
	a = basic:__getboth(),
	b = basic:__getonlyGet(),
}
do
	local i = basic:__getboth()
	local _shouldIncrement = false
	while true do
		if _shouldIncrement then
			i += 1
		else
			_shouldIncrement = true
		end
		if not (i < basic:__getboth() + 1) then
			break
		end
		print(i, list, record)
	end
end
if basic:__getboth() > 1 then
	print("big")
end
while basic:__getboth() > 100 do
	basic:__setboth(0)
end
local function usesDefault(a)
	if a == nil then
		a = basic:__getboth()
	end
	return a
end
print(usesDefault())
return {
	Basic = Basic,
}
