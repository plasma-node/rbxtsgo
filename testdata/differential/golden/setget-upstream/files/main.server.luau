-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("include"):WaitForChild("RuntimeLib"))
--/ <reference types="@rbxts/testez/globals" />
local ServerScriptService = TS.import(script, game:GetService("ReplicatedStorage"), "include", "node_modules", "@rbxts", "services").ServerScriptService
local TestEZ = TS.import(script, game:GetService("ReplicatedStorage"), "include", "node_modules", "@rbxts", "testez", "src")
local results = TestEZ.TestBootstrap:run({ ServerScriptService.tests })
if #results.errors > 0 or results.failureCount > 0 then
	error("Tests failed!")
end
