-- Says whether a file parses as Luau. Run: lune run parses.luau <file>
local fs = require("@lune/fs")
local luau = require("@lune/luau")
local process = require("@lune/process")
print(if pcall(luau.compile, fs.readFile(process.args[1])) then "parses" else "fails")
