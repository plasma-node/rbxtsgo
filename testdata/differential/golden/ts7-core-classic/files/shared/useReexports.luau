-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local _reexports = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "reexports")
local first = _reexports.default
local all = _reexports.all
local otherName = _reexports.otherName
local renamed = _reexports.renamed
local value = _reexports.value
local other = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "reexports").other
print(first(), all.other, otherName, renamed(), value, other)
return nil
