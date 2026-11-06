-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local Klass = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsClass")
local fn = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsFn")
local object = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsObject")
local Space = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsNamespace")
local nativeFn = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeFn")
local NativeClass = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeClass")
local defaulted = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport")
local unused = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsFn")
local Again = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsObject")
print(Klass.new().v, fn(), object.a, Space.run(), nativeFn(), NativeClass.new().v, defaulted.other, Again.b)
print(t)
return {
	Again = Again,
}
