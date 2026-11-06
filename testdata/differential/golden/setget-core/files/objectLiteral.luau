-- Compiled with roblox-ts v3.0.0
local store = 1
local literal = {
	plain = 1,
	__getvalue = function(self)
		return store
	end,
	__setvalue = function(self, val)
		store = val
	end,
	__getreadOnly = function(self)
		return "r"
	end,
	method = function(self)
		return self:__getvalue()
	end,
}
literal:__setvalue(2)
literal:__setvalue(literal:__getvalue() + 1)
literal:__setvalue(literal:__getvalue() + 1)
print(literal:__getvalue(), literal:__getreadOnly(), literal:method(), literal.plain)
local single = {
	__getone = function(self)
		return 1
	end,
}
print(single:__getone())
return nil
