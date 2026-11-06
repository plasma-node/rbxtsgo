-- Compiled with roblox-ts v3.0.0
local TS = _G[script]
local other = {
	other = "x",
}
local function parent()
	return TS.Promise.new(function(resolve)
		resolve(TS.import(script, script.Parent.Parent, "shapes"))
	end)
end
return {
	parent = parent,
	other = other,
}
