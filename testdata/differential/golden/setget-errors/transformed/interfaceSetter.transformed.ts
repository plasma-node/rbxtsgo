interface HasName {
    __setname(value: string);
}
declare const named: HasName;
named.__setname("m");
