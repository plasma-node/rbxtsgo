import data from "./data.json";
import * as starData from "./data.json";
import { key } from "./data.json";

print(data.key, starData.list, key);

async function load() {
	const lazy = await import("./defaultExport");
	const viaPaths = await import("shared/viaPaths");
	print(lazy.default(), lazy.other, viaPaths.value);
}
print(load);
