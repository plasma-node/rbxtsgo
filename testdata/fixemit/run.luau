-- Runs each module of ServerScriptService.ts.cases in a place built by Rojo, and prints what it prints. Each case runs on
-- its own, so one that fails doesn't hide the others. Run: lune run run.luau <place.rbxl>.
local fs = require("@lune/fs")
local luau = require("@lune/luau")
local process = require("@lune/process")
local roblox = require("@lune/roblox")
local task = require("@lune/task")

local game = roblox.deserializePlace(fs.readFile(process.args[1]))

-- RuntimeLib uses :WaitForChild(), and nothing here replicates.
roblox.implementMethod("Instance", "WaitForChild", function(self, ...)
	return self:FindFirstChild(...)
end)
-- Promise.lua indexes RunService.Heartbeat when it loads.
roblox.implementProperty("RunService", "Heartbeat", function()
	return {}
end, function() end)

local cache = {}
local robloxRequire

local function run(script)
	local environment = setmetatable({
		game = game,
		script = script,
		require = robloxRequire,
		task = task,
		tick = os.clock,
	}, { __index = roblox })
	return luau.load(luau.compile(script.Source), { debugName = script:GetFullName(), environment = environment })()
end

function robloxRequire(module)
	local path = module:GetFullName()
	if cache[path] == nil then
		cache[path] = table.pack(run(module))
	end
	return table.unpack(cache[path])
end

-- The location at the front of an error depends on how the script was loaded.
local function clean(message)
	return (tostring(message):gsub("^[^\n]-:%d+: ", ""))
end

local cases = game:GetService("ServerScriptService").ts.cases:GetChildren()
table.sort(cases, function(a, b)
	return a.Name < b.Name
end)
for _, case in cases do
	print("== " .. case.Name)
	local ok, err = xpcall(robloxRequire, function(message)
		return message
	end, case)
	if not ok then
		print("ERROR " .. clean(err))
	end
end
