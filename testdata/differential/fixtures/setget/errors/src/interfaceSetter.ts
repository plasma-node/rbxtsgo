interface HasName {
	set name(value: string); //! TS7010
}
declare const named: HasName;
named.name = "m";
