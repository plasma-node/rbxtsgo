// Deliberate errors, to compare diagnostics between tsc 5.9 and tsgo.
import { Players } from "@rbxts/services";
import { missing } from "shared/nope";
import { nothing } from "@rbxts/not-a-real-package";

const n: number = "not a number";
const arr: Array<string> = [1, 2];
Players.NotAProperty;
new Map<string, number>().set("a", "b");
const [a, b, c] = [1, 2] as [number, number];
print(missing, nothing, n, arr, a, b, c, undeclared);
