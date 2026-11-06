-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local exports = {}
exports.value = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultReExport").default
return exports
