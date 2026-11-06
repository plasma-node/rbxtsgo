package transformer

// Port of src/TSTransformer/nodes/class/transformClassLikeDeclaration.ts.

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/luau"
	"github.com/microsoft/TypeScript/tsc/rbxtsgo/project/shared"
)

const magicToStringMethod = "toString"

func createNameFunction(name string) *luau.FunctionExpression {
	return luau.Create(&luau.FunctionExpression{
		Statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
			Expression: luau.NewString(name),
		})),
		Parameters:   luau.NewList(),
		HasDotDotDot: false,
	})
}

func createBoilerplate(state *TransformState, node *ast.Node, className luau.AnyIdentifier, isClassExpression bool) *luau.List {
	isAbstract := ast.HasAbstractModifier(node)
	statements := luau.NewList()

	/* boilerplate:
	className = setmetatable({}, {
		__tostring = function() return "className" end;
	});
	className.__index = className;
	function className.new(...)
		local self = setmetatable({}, className);
		self:constructor(...);
		return self;
	end;
	function className:constructor()
	end;
	*/

	// 	className = setmetatable({}, {
	// 		__tostring = function() return "className" end;
	// 		__index = super,
	//	});

	// if a class is abstract and it does not extend any class, it can just be a plain table
	// otherwise we can use the default boilerplate
	extendsNode := getExtendsNode(node)
	if isAbstract && extendsNode == nil {
		statements.Push(luau.Create(&luau.Assignment{
			Left:     className,
			Operator: luau.AssignEqual,
			Right:    luau.NewMap(),
		}))
	} else {
		metatableFields := luau.NewList()
		var classNameText string
		if luau.IsTemporaryIdentifier(className) {
			classNameText = "Anonymous"
		} else {
			classNameText = className.(*luau.Identifier).Name
		}
		metatableFields.Push(luau.Create(&luau.MapField{
			Index: luau.Strings.Tostring,
			Value: createNameFunction(classNameText),
		}))

		if extendsNode != nil {
			extendsExp, extendsExpPrereqs := Capture(state, func() luau.Expression {
				return transformExpression(state, extendsNode.Expression())
			})
			superID := luau.NewID("super")
			statements.PushList(extendsExpPrereqs)
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  superID,
				Right: extendsExp,
			}))
			metatableFields.Push(luau.Create(&luau.MapField{
				Index: luau.Strings.Index,
				Value: superID,
			}))
		}

		metatable := luau.NewCall(luau.Globals.Setmetatable,
			luau.NewMap(),
			luau.Create(&luau.Map{Fields: metatableFields}),
		)

		if isClassExpression && node.Name() != nil {
			statements.Push(luau.Create(&luau.VariableDeclaration{
				Left:  transformIdentifierDefined(state, node.Name()),
				Right: metatable,
			}))
		} else {
			statements.Push(luau.Create(&luau.Assignment{
				Left:     className,
				Operator: luau.AssignEqual,
				Right:    metatable,
			}))
		}

		//	className.__index = className;
		statements.Push(luau.Create(&luau.Assignment{
			Left:     luau.NewProperty(className, "__index"),
			Operator: luau.AssignEqual,
			Right:    className,
		}))
	}

	// statements for className.new
	if !isAbstract {
		statementsInner := luau.NewList()

		//	local self = setmetatable({}, className);
		statementsInner.Push(luau.Create(&luau.VariableDeclaration{
			Left:  luau.Globals.Self,
			Right: luau.NewCall(luau.Globals.Setmetatable, luau.NewMap(), className),
		}))

		//	return self:constructor(...) or self;
		statementsInner.Push(luau.Create(&luau.ReturnStatement{
			Expression: luau.NewBinary(
				luau.Create(&luau.MethodCallExpression{
					Expression: luau.Globals.Self,
					Name:       "constructor",
					Args:       luau.MakeList(luau.Create(&luau.VarArgsLiteral{})),
				}),
				luau.BinaryOr,
				luau.Globals.Self,
			),
		}))

		//	function className.new(...)
		//	end;
		statements.Push(luau.Create(&luau.FunctionDeclaration{
			Name:         luau.NewProperty(className, "new"),
			Parameters:   luau.NewList(),
			HasDotDotDot: true,
			Statements:   statementsInner,
			Localize:     false,
		}))
	}

	return statements
}

func isClassHoisted(state *TransformState, node *ast.Node) bool {
	if node.Name() != nil {
		symbol := state.Checker.GetSymbolAtLocation(node.Name())
		if symbol == nil {
			panic(shared.AssertionMessage(""))
		}
		return state.IsHoisted[symbol] == true
	}
	return false
}

// transformClassLikeDeclarationResult is the object transformClassLikeDeclaration returns.
type transformClassLikeDeclarationResult struct {
	statements *luau.List
	name       luau.AnyIdentifier
}

func transformClassLikeDeclaration(state *TransformState, node *ast.Node) transformClassLikeDeclarationResult {
	isClassExpression := ast.IsClassExpression(node)
	statements := luau.NewList()

	isExportDefault := ast.HasSyntacticModifier(node, ast.ModifierFlagsExportDefault)

	if node.Name() != nil {
		validateIdentifier(state, node.Name())
	}

	/*
		local className;
		do
			OOP boilerplate
			class functions
		end
	*/

	shouldUseInternalName := isClassExpression && node.Name() != nil

	var returnVar luau.AnyIdentifier
	if shouldUseInternalName {
		returnVar = luau.NewTempID("class")
	} else if node.Name() != nil {
		returnVar = transformIdentifierDefined(state, node.Name())
	} else if isExportDefault {
		returnVar = luau.NewID("default")
	} else {
		returnVar = luau.NewTempID("class")
	}

	var internalName luau.AnyIdentifier
	if shouldUseInternalName {
		internalName = transformIdentifierDefined(state, node.Name())
	} else {
		internalName = returnVar
	}
	state.ClassIdentifierMap[node] = internalName
	if !isClassHoisted(state, node) {
		statements.Push(luau.Create(&luau.VariableDeclaration{
			Left:  returnVar,
			Right: nil,
		}))
	}

	// OOP boilerplate + class functions
	statementsInner := luau.NewList()
	statementsInner.PushList(createBoilerplate(state, node, internalName, isClassExpression))

	constructor := findConstructor(node)
	if constructor != nil {
		statementsInner.PushList(transformClassConstructor(state, constructor, internalName))
	} else {
		statementsInner.PushList(transformImplicitClassConstructor(state, node, internalName))
	}

	for _, member := range node.Members() {
		if (ast.IsPropertyDeclaration(member) || ast.IsMethodDeclaration(member)) &&
			(ast.IsIdentifier(member.Name()) || ast.IsStringLiteral(member.Name())) &&
			luau.IsReservedClassField(member.Name().Text()) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoReservedClassFields.At(shared.AtNode(member.Name())))
		}
		if ast.IsAutoAccessorPropertyDeclaration(member) {
			// member must have AccessorKeyword to be AutoAccessorPropertyDeclaration
			var keyword *ast.Node
			for _, m := range member.ModifierNodes() {
				if m.Kind == ast.KindAccessorKeyword {
					keyword = m
					break
				}
			}
			state.Diagnostic.AddDiagnostic(shared.Errors.NoAutoAccessorModifiers.At(shared.AtNode(keyword)))
		}
	}

	var methods []*ast.Node
	var staticDeclarations []*ast.Node

	for _, member := range node.Members() {
		validateMethodAssignment(state, member)
		if ast.IsConstructorDeclaration(member) ||
			ast.IsIndexSignatureDeclaration(member) ||
			ast.IsSemicolonClassElement(member) {
			continue
		} else if ast.IsMethodDeclaration(member) {
			methods = append(methods, member)
		} else if ast.IsPropertyDeclaration(member) {
			// non-static properties are done in transformClassConstructor
			if !ast.HasStaticModifier(member) {
				continue
			}
			staticDeclarations = append(staticDeclarations, member)
		} else if ast.IsAccessor(member) {
			state.Diagnostic.AddDiagnostic(shared.Errors.NoGetterSetter.At(shared.AtNode(member)))
		} else if ast.IsClassStaticBlockDeclaration(member) {
			staticDeclarations = append(staticDeclarations, member)
		} else {
			panic(shared.AssertionMessage("ClassMember kind not implemented: " + getKindName(member.Kind)))
		}
	}

	classType := state.Checker.GetTypeOfSymbolAtLocation(node.Symbol(), node)
	instanceType := state.Checker.GetDeclaredTypeOfSymbol(node.Symbol())

	for _, method := range methods {
		if ast.IsIdentifier(method.Name()) || ast.IsStringLiteral(method.Name()) {
			if luau.IsMetamethod(method.Name().Text()) {
				state.Diagnostic.AddDiagnostic(shared.Errors.NoClassMetamethods.At(shared.AtNode(method.Name())))
			}

			if ast.HasStaticModifier(method) {
				if state.Checker.GetPropertyOfType(instanceType, method.Name().Text()) != nil {
					state.Diagnostic.AddDiagnostic(shared.Errors.NoInstanceMethodCollisions.At(shared.AtNode(method)))
				}
			} else {
				if state.Checker.GetPropertyOfType(classType, method.Name().Text()) != nil {
					state.Diagnostic.AddDiagnostic(shared.Errors.NoStaticMethodCollisions.At(shared.AtNode(method)))
				}
			}
		}

		statements, prereqs := Capture(state, func() *luau.List {
			return transformMethodDeclaration(state, method, &mapPointer{name: "name", value: internalName})
		})
		statementsInner.PushList(prereqs)
		statementsInner.PushList(statements)
	}

	toStringProperty := state.Checker.GetPropertyOfType(instanceType, magicToStringMethod)
	if toStringProperty != nil && toStringProperty.Flags()&ast.SymbolFlagsMethod != 0 {
		statementsInner.Push(luau.Create(&luau.MethodDeclaration{
			Expression:   internalName,
			Name:         "__tostring",
			HasDotDotDot: false,
			Parameters:   luau.NewList(),
			Statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
				Expression: luau.Create(&luau.MethodCallExpression{
					Expression: luau.Globals.Self,
					Name:       magicToStringMethod,
					Args:       luau.NewList(),
				}),
			})),
		}))
	}

	for _, declaration := range staticDeclarations {
		if ast.IsClassStaticBlockDeclaration(declaration) {
			// Body() has no case for a static block in TS 7, so read the field
			statementsInner.PushList(transformBlock(state, declaration.AsClassStaticBlockDeclaration().Body))
		} else {
			statements, prereqs := Capture(state, func() *luau.List {
				return transformPropertyDeclaration(state, declaration, internalName)
			})
			statementsInner.PushList(prereqs)
			statementsInner.PushList(statements)
		}
	}

	// if using internal name, assign to return var
	if shouldUseInternalName {
		statementsInner.Push(luau.Create(&luau.Assignment{
			Left:     returnVar,
			Operator: luau.AssignEqual,
			Right:    internalName,
		}))
	}

	statementsInner.PushList(transformDecorators(state, node, returnVar))

	statements.Push(luau.Create(&luau.DoStatement{
		Statements: statementsInner,
	}))

	return transformClassLikeDeclarationResult{statements: statements, name: returnVar}
}

// getAllSuperTypeNodes is ts.getAllSuperTypeNodes from TS 5.5.3, used by validateMethodAssignment. In JS files, TS 5.5.3
// reads JSDoc tags here. TS 7's reparser already puts `@implements` in the implements clause, so that case is covered.
// For `@augments`, TS 5.5.3 returns the tag's class node and this returns the extends element. Only allowJs reaches that.
func getAllSuperTypeNodes(node *ast.Node) []*ast.Node {
	if ast.IsInterfaceDeclaration(node) {
		return ast.GetHeritageElements(node, ast.KindExtendsKeyword)
	}
	if ast.IsClassLike(node) {
		var result []*ast.Node
		if baseType := ast.GetClassExtendsHeritageElement(node); baseType != nil {
			result = append(result, baseType)
		}
		return append(result, ast.GetHeritageElements(node, ast.KindImplementsKeyword)...)
	}
	return nil
}
