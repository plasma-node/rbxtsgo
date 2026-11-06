import { Shape } from "./shapes";
import { Regular } from "../util/helper";
import type { Union } from "../util/helper";
import unrelated = require("./deep/other");
export declare function inferred(): import("./shapes").Square;
export declare const viaHelper: number;
export declare const fromRequire: unrelated.Other;
export declare function usesInstance(): Part;
export declare function usesFolder(parent: Instance, name: Regular): {
    folder: Folder | undefined;
    name: Regular;
};
export declare class Model implements Shape {
    sides: number;
    union?: Union;
    shape(): Shape;
}
export type Lazy = import("./deep/other").Other;
export type LazyQualified = typeof import("./deep/other");
