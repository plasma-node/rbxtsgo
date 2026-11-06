-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local data = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "data")
local starData = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "data")
local key = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "data").key
print(data.key, starData.list, key)
local load = TS.async(function()
	local lazy = TS.await(TS.Promise.new(function(resolve)
		resolve(TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport"))
	end))
	local viaPaths = TS.await(TS.Promise.new(function(resolve)
		resolve(TS.import(script, game:GetService("ReplicatedStorage"), "TS", "viaPaths"))
	end))
	print(lazy.default(), lazy.other, viaPaths.value)
end)
print(load)
return nil
