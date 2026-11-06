import { Players, ReplicatedStorage, Workspace } from "@rbxts/services";
import { t } from "@rbxts/t";
import { Janitor } from "@rbxts/janitor";
import { atom, effect } from "@rbxts/charm";
import type { Profile, UserId } from "shared/types";
import { DEFAULT_COINS } from "shared/types";
import { Counter, divmod, firstOf, Flag, NamedCounter, Pair, Point, Team } from "shared/util";
// arrays
const scores = [1, 2, 3];
scores.push(4);
const doubled = scores.map(v => v * 2);
const total = doubled.size();
const readonlyScores: ReadonlyArray<number> = scores;
const head = firstOf(readonlyScores);
// maps and sets
const profiles = new Map<UserId, Profile>();
profiles.set(1, { id: 1, coins: DEFAULT_COINS });
const profile = profiles.get(1);
const readonlyProfiles: ReadonlyMap<UserId, Profile> = profiles;
const tags = new Set<string>();
tags.add("vip");
// tuples and LuaTuple
const pair: Pair = ["a", 1];
const [pairName, pairScore] = pair;
const [quotient, remainder] = divmod(7, 2);
const luaTuple = divmod(9, 4);
const [found, foundEnd] = string.find("hello", "ll");
// destructuring assignment
let px = 0;
let py = 0;
const origin: Point = { x: 0, y: 0 };
({ x: px, y: py } = origin);
[px, py] = [1, 2];
const shorthand = { px, py };
// enums
const team = Team.Blue;
const flag = Flag.On;
const robloxKey = Enum.KeyCode.E;
// Roblox services and instances
const part = new Instance("Part");
part.Parent = Workspace;
const localCount = Players.GetPlayers().size();
const folder = ReplicatedStorage.FindFirstChild("Shared");
const signalConn = Players.PlayerAdded.Connect(player => {
    print(player.Name, player.UserId);
});
// macros and macro-like identifiers
const isNumber = typeIs(total, "number");
const kind = typeOf(part);
const asserted = identity<number>(5);
for (const i of $range(1, 3)) {
    print(i);
}
const later = Promise.resolve(1);
// optional access, generics, classes
const coinsOrNone = profile?.coins;
const counter = new NamedCounter("hits");
counter.add("x");
const genericCounter: Counter<string> = counter;
// object literal context
const callbacks: {
    onDone: (ok: boolean) => void;
} = {
    onDone(ok) {
        print(ok);
    },
};
// third-party packages
const check = t.strictInterface({ id: t.number });
const janitor = new Janitor();
janitor.Add(signalConn, "Disconnect");
const coins = atom(0);
effect(() => print(coins()));
export { pairName, pairScore, quotient, remainder, luaTuple, found, foundEnd, shorthand };
export { team, flag, robloxKey, localCount, folder, isNumber, kind, asserted, later };
export { coinsOrNone, genericCounter, callbacks, check, tags, readonlyProfiles, head };
export type { Profile };
// iteration: these all rely on Symbol.iterator or iterable types
function* countUp(limit: number) {
    for (const i of $range(1, limit)) {
        yield i;
    }
}
for (const v of scores) {
    print(v);
}
for (const [key, value] of profiles) {
    print(key, value);
}
for (const tag of tags) {
    print(tag);
}
for (const n of countUp(3)) {
    print(n);
}
for (const [word] of "a b".gmatch("%a")) {
    print(word);
}
for (const [index, value] of pairs(scores)) {
    print(index, value);
}
const spreadSet = [...tags];
const [firstTag] = tags;
const spreadMap = [...profiles];
export { spreadSet, firstTag, spreadMap };
// literal and special symbols
const declaredTrue: true = true;
const declaredFalse: false = false;
const nothing = undefined;
function legacyArguments() {
    // eslint-disable-next-line prefer-rest-params
    return arguments;
}
export { declaredTrue, declaredFalse, nothing, legacyArguments };
