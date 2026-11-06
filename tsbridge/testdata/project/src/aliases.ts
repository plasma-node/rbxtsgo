import defaultUsed, { namedInDefaultFile as renamedUnused } from "./lib";
import defaultUnused from "./lib";
import { OnlyType, unusedValue, usedValue, UsedAsTypeOnly } from "./lib";
import type { UsedAsTypeOnly as ExplicitTypeOnly } from "./lib";
import { type OnlyType as InlineTypeOnly, usedValue as usedAgain } from "./lib";
import * as namespaceUsed from "./lib";
import * as namespaceUnused from "./lib";
import { viaBaseUrl } from "shared/viaBaseUrl";
import { fromDeep } from "@rbxts/exportspkg/lib/deep";
import { fine } from "@rbxts/brokenpkg";

const a: OnlyType = { a: usedValue + usedAgain + namespaceUsed.usedValue + defaultUsed() };
const b: UsedAsTypeOnly | ExplicitTypeOnly | InlineTypeOnly | undefined = undefined;
export { a, b, viaBaseUrl, fromDeep, fine };
