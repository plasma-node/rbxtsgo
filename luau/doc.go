// Package luau is a Go port of @roblox-ts/luau-ast 2.0.0: the Luau AST used by
// the roblox-ts compiler, plus the renderer that turns it into Luau source.
//
// Upstream: https://github.com/roblox-ts/luau-ast at commit
// ef18d8bccbe5a31435ab56683a81817a7f22ac51 (tag 2.0.0, the npm gitHead of
// @roblox-ts/luau-ast@2.0.0). Output must match the JS renderer byte for byte.
//
// # File map
//
//	Go file               upstream source (src/...)
//	kinds.go              LuauAST/impl/enums.ts, LuauAST/util/getKindName.ts
//	operators.go          LuauAST/types/operators.ts
//	nodes.go              LuauAST/types/nodes.ts, LuauAST/types/mapping.ts
//	create.go             LuauAST/impl/create.ts
//	list.go               LuauAST/impl/List.ts
//	typeguards.go         LuauAST/impl/typeGuards.ts
//	globals.go            LuauAST/impl/globals.ts
//	strings.go            LuauAST/impl/strings.ts
//	util.go               LuauAST/util/{assert,isMetamethod,isReservedClassField,
//	                      isReservedIdentifier,isValidIdentifier,isValidNumberLiteral}.ts
//	render.go             LuauRenderer/render.ts
//	render_state.go       LuauRenderer/RenderState.ts
//	solve_temp_ids.go     LuauRenderer/solveTempIds.ts
//	visit.go              LuauRenderer/util/visit.ts
//	render_util.go        LuauRenderer/util/{getEnding,getSafeBracketEquals,needsParentheses,
//	                      renderArguments,renderParameters,renderStatements}.ts
//	render_indexable.go   LuauRenderer/nodes/expressions/indexable/*.ts
//	render_expressions.go LuauRenderer/nodes/expressions/*.ts
//	render_statements.go  LuauRenderer/nodes/statements/*.ts
//	render_fields.go      LuauRenderer/nodes/fields/*.ts
//	jsnum.go              no upstream file: ECMAScript Number() and String(number)
//
// # Naming map (upstream to Go)
//
//	luau.create(kind, fields)   Create(&Kind{...})
//	luau.tempId, none, nil, bool, number, string, id, comment, array, set, map,
//	mixedTable, binary, unary, property, call
//	                            NewTempID, NewNone, NewNil, NewBool, NewNumber, NewString,
//	                            NewID, NewComment, NewArray, NewSet, NewMap, NewMixedTable,
//	                            NewBinary, NewUnary, NewProperty, NewCall
//	luau.list.x(list, ...)      methods on *List (list.Push(v)), plus MakeList and NewList
//	luau.globals, luau.strings  Globals, Strings
//	luau.SyntaxKind.X           KindX
//	luau.isX(node)              IsX(node)
//	render, renderAST, renderStatements, solveTempIds, RenderState
//	                            Render, RenderAST, RenderStatements, SolveTempIDs, RenderState
//
// # Semantics preserved on purpose
//
// Nodes are mutable pointers that carry a parent pointer. Create adopts its
// children: a child that already has a parent is shallow-cloned (struct copy,
// so lists and grandchildren stay shared) and the clone is stored in the new
// node; a child without a parent is adopted in place. List helpers mutate and
// share list nodes exactly like upstream (PushList marks the other list
// read-only but leaves its head and tail pointing into the merged chain, Shift
// does not clear the removed node's Next, and so on). The transformer depends
// on this, so do not "fix" it.
//
// Fields are adopted in the order they are declared in nodes.go. Upstream uses
// the property order of the object literal at each call site. This only
// matters when the same parentless node appears in two fields of one Create
// call, and in that case both orders render identically.
//
// # Concurrency
//
// Globals and Strings are shared mutable nodes, as upstream: the first Create
// that uses one adopts it in place, later uses clone it. Concurrent Create
// calls that use them race on that parent pointer. A parallel transformer
// should either serialise first use (use each global once at start-up) or
// clone with ShallowClone before use.
package luau
