import { fromLua } from "util/native";
export const value = fromLua();
export type Fn = typeof fromLua;
