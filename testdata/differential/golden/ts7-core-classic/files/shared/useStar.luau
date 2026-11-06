-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local object = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsObject")
local Space = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsNamespace")
local star = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport")
local asDefault = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultAsName")
local nativeDefault = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeDefault")
print(object.a, Space.run(), star.default(), star.other, asDefault.alias, nativeDefault.default)
return nil
