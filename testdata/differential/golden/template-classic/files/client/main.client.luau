-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
local Players = TS.import(script, game:GetService("ReplicatedStorage"), "rbxts_include", "node_modules", "@rbxts", "services").Players
local makeHello = TS.import(script, game:GetService("ReplicatedStorage"), "TS", "module").makeHello
print(makeHello(`main.client.ts, as {Players.LocalPlayer.Name}`))
