/// <reference types="@rbxts/types" />
import { Shape, makeSquare } from 'models/shapes';
import { helper as help, Regular } from "../util/helper";
import type { Union } from "util/helper";
import unrelated = require("models/deep/other");

// The return type here names a type from another module without importing it, so the declaration has import("...").
export function inferred() {
	return makeSquare();
}
export const viaHelper = help(1);
export const fromRequire = unrelated.other;
export function usesInstance() {
	return new Instance("Part");
}
export function usesFolder(parent: Instance, name: Regular) {
	return { folder: parent.FindFirstChild("x") as Folder | undefined, name };
}
export class Model implements Shape {
	public sides = 3;
	public union?: Union;
	public shape(): Shape {
		return this;
	}
}
export type Lazy = import("models/deep/other").Other;
export type LazyQualified = typeof import("models/deep/other");
