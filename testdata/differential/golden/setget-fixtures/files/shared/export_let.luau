-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local exports = {}
exports.counter = 0
local function bump()
	exports.counter += 1
	exports.counter += 1
	return exports.counter
end
for _k, _v in TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exports_order") or {} do
	exports[_k] = _v
end
exports.all = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "strings_unicode")
exports.firstAgain = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exports_order").first
exports.bump = bump
return exports
