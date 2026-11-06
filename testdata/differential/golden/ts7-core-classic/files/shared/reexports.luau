-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local exports = {}
for _k, _v in TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport") or {} do
	exports[_k] = _v
end
exports.all = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport")
exports.default = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport").default
local _defaultExport = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport")
exports.renamed = _defaultExport.default
exports.otherName = _defaultExport.other
exports.value = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "viaPaths").value
return exports
