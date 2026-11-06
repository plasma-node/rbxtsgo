-- Checks each file of the string matrix: does it parse, and is its value what JS says. One line per case.
-- Run: lune run check.luau <project folder> <output folder name>
local fs = require("@lune/fs")
local luau = require("@lune/luau")
local process = require("@lune/process")
local serde = require("@lune/serde")

local dir = process.args[1]
local expected = serde.decode("json", fs.readFile(dir .. "/expected.json"))
local names = {}
for name in expected do
	table.insert(names, name)
end
table.sort(names)

local function hex(text)
	return (text:gsub(".", function(c)
		return string.format("%02x", c:byte())
	end))
end

for _, name in names do
	local source = fs.readFile(dir .. "/" .. process.args[2] .. "/" .. name .. ".luau")
	local ok, bytecode = pcall(luau.compile, source)
	if not ok then
		print(name, "parse-error")
	else
		local run = luau.load(bytecode, {
			environment = {
				print = function() end,
				tostring = tostring,
			},
		})
		print(name, if hex(run()) == expected[name] then "ok" else "wrong")
	end
end
