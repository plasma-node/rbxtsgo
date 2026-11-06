-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local _crossA = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "crossA")
local Fallback = _crossA.default
local literalExport = _crossA.literalExport
local Shared = _crossA.Shared
local sharedInstance = _crossA.sharedInstance
local all = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "crossA")
local s = Shared.new()
s:__setdata(1)
s:__setdata(s:__getdata() + 1)
print(s:__getdata(), sharedInstance:__getdata(), literalExport:__gety())
print(all.sharedInstance:__getdata(), all.Shared.new():__getdata())
local f = Fallback.new()
print(f:__getextra(), f:__getdata())
return nil
