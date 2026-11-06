// TS 5.5 widens the argument type in the message when the parameter is never. TS 7 prints the literal.
declare function get(key: never): void;
get("Update");
export {};
