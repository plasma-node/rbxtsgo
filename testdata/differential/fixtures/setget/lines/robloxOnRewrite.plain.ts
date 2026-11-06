class D {
	// A comment.

	public __getloose(): any {
		return 1;
	}
}

const d = new D();


print(d.__getloose().field);


const anything: any = 1;


print(anything.field);
