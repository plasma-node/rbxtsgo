"use strict";
// Serializes a live @roblox-ts/luau-ast object graph to JSON, keeping object
// identity (parent pointers, shared nodes and lists, raw list-node links).
// The Go tests rebuild the exact same graph from it, so the renderer sees the
// same shared and cloned structure the JS renderer saw.
//
// Reusable on its own: a patched upstream compiler can call
// `serialize({ ast: list })` and write the result next to the rendered Luau to
// make extra golden cases (see README.md).
//
// Format: { nodes: [...], lists: [...], listNodes: [...], roots: { name: ref } }
//   node ref     { "$n": index into nodes }
//   list ref     { "$l": index into lists }
//   list node ref  plain integer index into listNodes, or null
// Ids are assigned breadth first from the roots, visiting properties in
// sorted key order, so the output is deterministic.

const luau = require("@roblox-ts/luau-ast").default;

// Names of luau.SyntaxKind values, in enum order. Written out here instead of
// using luau.SyntaxKind[kind], whose reverse mapping returns alias names for
// some values.
const KIND_NAMES = [
	"Identifier", "TemporaryIdentifier", "ComputedIndexExpression", "PropertyAccessExpression", "CallExpression",
	"MethodCallExpression", "ParenthesizedExpression", "None", "NilLiteral", "FalseLiteral", "TrueLiteral",
	"NumberLiteral", "StringLiteral", "VarArgsLiteral", "FunctionExpression", "BinaryExpression", "UnaryExpression",
	"IfExpression", "InterpolatedString", "Array", "Map", "Set", "MixedTable", "Assignment", "BreakStatement",
	"CallStatement", "ContinueStatement", "DoStatement", "WhileStatement", "RepeatStatement", "IfStatement",
	"NumericForStatement", "ForStatement", "FunctionDeclaration", "MethodDeclaration", "VariableDeclaration",
	"ReturnStatement", "Comment", "MapField", "InterpolatedStringPart",
];

for (let i = 0; i < KIND_NAMES.length; i++) {
	if (luau.SyntaxKind[KIND_NAMES[i]] !== i) throw new Error(`kind table out of sync at ${i}`);
}

function serialize(roots) {
	const nodes = [];
	const lists = [];
	const listNodes = [];
	const nodeIds = new Map();
	const listIds = new Map();
	const listNodeIds = new Map();
	const queue = [];

	function nodeRef(node) {
		let id = nodeIds.get(node);
		if (id === undefined) {
			id = nodes.length;
			nodeIds.set(node, id);
			nodes.push(null);
			queue.push(["n", id, node]);
		}
		return { $n: id };
	}
	function listRef(list) {
		let id = listIds.get(list);
		if (id === undefined) {
			id = lists.length;
			listIds.set(list, id);
			lists.push(null);
			queue.push(["l", id, list]);
		}
		return { $l: id };
	}
	function listNodeRef(listNode) {
		if (listNode === undefined) return null;
		let id = listNodeIds.get(listNode);
		if (id === undefined) {
			id = listNodes.length;
			listNodeIds.set(listNode, id);
			listNodes.push(null);
			queue.push(["ln", id, listNode]);
		}
		return id;
	}
	function ref(value) {
		if (luau.list.isList(value)) return listRef(value);
		if (luau.isNode(value)) return nodeRef(value);
		throw new Error("not a node or list: " + String(value));
	}

	const rootRefs = {};
	for (const name of Object.keys(roots)) rootRefs[name] = ref(roots[name]);

	for (let q = 0; q < queue.length; q++) {
		const [type, id, obj] = queue[q];
		if (type === "n") {
			const out = {};
			for (const key of Object.keys(obj).sort()) {
				const value = obj[key];
				if (value === undefined) continue;
				if (key === "kind") {
					out.kind = KIND_NAMES[value];
				} else if (key === "parent" || luau.isNode(value) || luau.list.isList(value)) {
					out[key] = ref(value);
				} else if (["string", "number", "boolean"].includes(typeof value)) {
					out[key] = value;
				} else {
					throw new Error(`unexpected property ${key}: ${String(value)}`);
				}
			}
			nodes[id] = out;
		} else if (type === "l") {
			const out = {};
			out.head = listNodeRef(obj.head);
			out.readonly = obj.readonly;
			out.tail = listNodeRef(obj.tail);
			lists[id] = out;
		} else {
			const out = {};
			out.next = listNodeRef(obj.next);
			out.prev = listNodeRef(obj.prev);
			out.value = nodeRef(obj.value);
			listNodes[id] = out;
		}
	}

	return { nodes, lists, listNodes, roots: rootRefs };
}

module.exports = { serialize, KIND_NAMES };
