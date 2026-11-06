export let counter = 0;
export function bump() {
    counter++;
    counter += 1;
    return counter;
}
export * from "./exports_order";
export * as all from "./strings_unicode";
export { first as firstAgain } from "./exports_order";
