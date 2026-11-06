-- Compiled with roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("rbxts_include"):WaitForChild("RuntimeLib"))
-- Function declarations after a final statement. Issues #2847 and #2098, open upstream with no fix, so nothing here is
-- ported. In JS the function exists from the start of its block, and sees every binding in the block.
-- #2098
local function calledByTheReturn()
	local function bar()
		return "Hello, world!"
	end
	return bar()
end
-- #2847
local function assignsAnOuterLocal()
	local variable
	local function SetVariable(value)
		variable = value
	end
	variable = 0
	SetVariable(2)
	return variable
end
local function calledBeforeLocalsItReads()
	local later
	local function readLater()
		return if type(later) == "nil" then "unset" else "set"
	end
	local before = readLater()
	later = 5
	local after = readLater()
	return `{before} {after} {later}`
end
local function readsDestructuredLocal()
	local a, b, c
	local function sum()
		return a + b + c
	end
	local _binding = { 1, 2 }
	a = _binding[1]
	b = _binding[2]
	local _binding_1 = {
		c = 3,
	}
	c = _binding_1.c
	return sum()
end
local function mutualRecursion()
	local isOdd
	local function isEven(n)
		return if n == 0 then true else isOdd(n - 1)
	end
	function isOdd(n)
		return if n == 0 then false else isEven(n - 1)
	end
	return isEven(4) and not isEven(3)
end
local function callsAFunctionBeforeTheReturn()
	local function late()
		return 41
	end
	local function early()
		return late() + 1
	end
	return early()
end
local function readsFunctionAsAValue()
	local function valued()
		return "value"
	end
	local callback = valued
	return callback()
end
local function laterStatementsStayDropped()
	local order
	local function visible()
		table.insert(order, "in function")
		return "done"
	end
	order = {}
	table.insert(order, "before")
	local result = visible()
	return `{table.concat(order, ",")} {result}`
end
local function classAfterReturnStaysDropped()
	local function helper()
		return "class dropped"
	end
	return helper()
end
local function insideIfBlock(flag)
	if flag then
		local function inner()
			return "in block"
		end
		return inner()
	end
	return "skipped"
end
local function afterContinueInLoop()
	local visited = {}
	for _, value in { 1, 2 } do
		local function double(n)
			return n * 2
		end
		local _arg0 = double(value)
		table.insert(visited, _arg0)
		continue
	end
	return table.concat(visited, ",")
end
local function afterBreakInLoop()
	local result = 0
	while true do
		local function three()
			return 3
		end
		result = three()
		break
	end
	return result
end
local function afterBreakInCase(value)
	repeat
		if value == 1 then
			local function inCase()
				return "in case"
			end
			print("case", inCase())
			break
		end
	until true
	return value
end
local function overloads()
	local function overloaded(a)
		return a
	end
	return overloaded(1)
end
local function capturesParameter(parameter)
	local function getParameter()
		return parameter
	end
	return getParameter()
end
local function generatorBody()
	local function numbers()
		return TS.generator(function()
			local function one()
				return 1
			end
			coroutine.yield(one())
			return 2
		end)
	end
	local iterator = numbers()
	return iterator.next().value
end
local function nestedAfterReturn()
	local function outer()
		local function inner()
			return "nested"
		end
		return inner()
	end
	return outer()
end
local function usedLaterThanTheCall()
	local count
	local function increment()
		count += 1
	end
	count = 0
	increment()
	increment()
	return count
end
print("#2098", calledByTheReturn())
print("#2847", assignsAnOuterLocal())
print("destructured", readsDestructuredLocal())
print("mutual", mutualRecursion())
print("before the return", callsAFunctionBeforeTheReturn())
print("as a value", readsFunctionAsAValue())
print("dropped", laterStatementsStayDropped())
print("class", classAfterReturnStaysDropped())
print("if", insideIfBlock(true), insideIfBlock(false))
print("continue", afterContinueInLoop())
print("break", afterBreakInLoop())
print("case", afterBreakInCase(1))
print("overloads", overloads())
print("parameter", capturesParameter(7))
print("generator", generatorBody())
print("nested", nestedAfterReturn())
print("count", usedLaterThanTheCall())
return calledBeforeLocalsItReads
