-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local exports = {}
local _exportEqualsClass = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsClass")
exports.Class = _exportEqualsClass
exports.ClassAlias = _exportEqualsClass
exports.callback = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsFunction")
local _exportEqualsObject = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsObject")
exports.default = _exportEqualsObject
exports.objectValue = _exportEqualsObject.value
exports.DefaultClass = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultClass").default
return exports
