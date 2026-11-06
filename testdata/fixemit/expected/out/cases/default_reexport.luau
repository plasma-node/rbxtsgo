-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
-- Upstream PR #3115's runtime tests, in tests/src/tests/moduleSyntax.spec.ts, printing what they compare. Issue #2799.
-- Its string-literal export names are left out, because TS 5.5.3, which JS 3.0.0 runs on, can't parse them.
local _defaultReExport = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultReExport")
local ExportedDefaultClass = _defaultReExport.DefaultClass
local callback = _defaultReExport.callback
local Class = _defaultReExport.Class
local ClassAlias = _defaultReExport.ClassAlias
local objectValue = _defaultReExport.objectValue
local defaultReExport = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultReExport").default
local chainedReExport = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultReExportChain").value
local DefaultClass = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "defaultClass").default
local ExportEqualsClass = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsClass")
local exportEqualsFunction = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsFunction")
local exportEqualsObject = TS.import(script, game:GetService("ServerScriptService"), "ts", "lib", "exportEqualsObject")
print("class", Class == ExportEqualsClass)
print("class alias", ClassAlias == ExportEqualsClass)
print("function", callback == exportEqualsFunction)
print("object", defaultReExport == exportEqualsObject)
print("chain", chainedReExport == exportEqualsObject)
print("named property", objectValue)
print("ES default", ExportedDefaultClass == DefaultClass)
print("ES default value", ExportedDefaultClass.new().value)
local _result = callback
if _result ~= nil then
	_result = _result()
end
print("called", _result)
print("constructed", if Class ~= nil then Class.new().value else "nothing")
return nil
