-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local Klass = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsClass")
local fn = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsFn")
local object = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsObject")
local Space = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "exportEqualsNamespace")
local nativeFn = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeFn")
local NativeClass = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeClass")
local nativeDefault = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "nativeDefault").default
local _defaultExport = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultExport")
local named = _defaultExport.default
local other = _defaultExport.other
local _defaultAsName = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "defaultAsName")
local asDefault = _defaultAsName.default
local alias = _defaultAsName.alias
print(Klass.new().v, fn(), object.a, Space.run(), nativeFn(), NativeClass.new().v, nativeDefault)
print(named(), other, asDefault, alias)
return nil
