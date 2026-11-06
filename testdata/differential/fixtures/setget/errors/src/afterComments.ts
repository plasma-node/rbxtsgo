/*
 * A comment that stays.
 */



// Another comment.



const fine = 1;



// An error far down the file, with text before it on the line.
print(fine); const wrong: string = 1; //! TS2322


class C {
	public get é() { return 1; }
}
/* 😀 */ const alsoWrong: string = new C().é; //! TS2322
print(wrong, alsoWrong);
