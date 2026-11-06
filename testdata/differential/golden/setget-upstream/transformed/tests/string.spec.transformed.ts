export = () => {
    it("should support string methods", () => {
        expect("Hello, world".sub(1, 1)).to.equal("H");
    });
    it("should support string methods on identifiers", () => {
        const str = "Hello, world";
        expect(str.sub(1, 1)).to.equal("H");
    });
    it("should support string.split", () => {
        function checkLen<T>(len: number, arr: Array<T>) {
            expect(arr.size()).to.equal(len);
            return arr;
        }
        const str = "Hello, world";
        const chars = str.byte(0, -1).map(i => string.char(i));
        const words = ["Hello", "world"];
        const hSplit = ["", "ello, world"];
        expect(str.split("").every((char, i) => char === chars[i])).to.equal(true);
        expect(str.split(", ").every((word, i) => word === words[i])).to.equal(true);
        expect(str.split("H").every((word, i) => word === hSplit[i])).to.equal(true);
        expect(checkLen(1, "".split("a"))[0]).to.equal("");
        for (let i = 2; i < 10; i++) {
            const str = "d".rep(i - 1);
            const str1 = str.split("d");
            expect(str1.size()).to.equal(i);
            expect(str1.every(c => c === "")).to.equal(true);
        }
        expect("".split("").isEmpty()).to.equal(true);
        const slasher = ["", "Validark", "Osyris", "Vorlias", ""];
        expect(checkLen(5, "/Validark/Osyris/Vorlias/".split("/")).every((word, i) => word === slasher[i])).to.equal(true);
        expect(checkLen(4, "Validark/Osyris/Vorlias/".split("/")).every((word, i) => word === slasher[i + 1])).to.equal(true);
        expect(checkLen(3, "Validark/Osyris/Vorlias".split("/")).every((word, i) => word === slasher[i + 1])).to.equal(true);
    });
    it("should support calling gmatch", () => {
        expect("Hello".gmatch(".")()[0]).to.equal("H");
    });
    it("should support the spread operator on strings", () => {
        const array4 = ["H", "i", "y", "a"];
        expect([..."Hiya"].every((x, i) => x === array4[i])).to.equal(true);
    });
    it("should support string.find", () => {
        const data = "Hello".find("H", 1, true);
        if (data[0]) {
            expect(data[0]).to.equal(1);
            expect(data[1]).to.equal(1);
        }
        const data2 = "Hello".find("e", 2, true);
        if (data2[0]) {
            expect(data2[0]).to.equal(2);
            expect(data2[1]).to.equal(2);
        }
    });
    it("should support concatenating strings", () => {
        expect("a" + 1 + true + false).to.equal("a1truefalse");
    });
    it("should support variable string indices", () => {
        let i = 1;
        let j = 3;
        expect("foobar".sub(i, j)).to.equal("foo");
    });
    it("should support proper destructuring and iterating", () => {
        function compare(results: Array<string>, array2: Array<string>) {
            for (let i = 0; i < array2.size(); i++) {
                expect(results[i]).to.equal(array2[i]);
            }
        }
        // optimized destructuring
        compare([..."𝟘𝟙𝟚𝟛"], ["𝟘", "𝟙", "𝟚", "𝟛"]);
        compare([..."யாமறிந்த"], ["ய", "ா", "ம", "ற", "ி", "ந", "்", "த"]);
        const spreadString = (str: string) => [...str];
        // run-time destructuring
        compare(spreadString("𝟘𝟙𝟚𝟛"), ["𝟘", "𝟙", "𝟚", "𝟛"]);
        compare(spreadString("யாமறிந்த"), ["ய", "ா", "ம", "ற", "ி", "ந", "்", "த"]);
        let i = 0;
        for (const substr of "𝟘𝟙𝟚𝟛") {
            expect(substr).to.equal(["𝟘", "𝟙", "𝟚", "𝟛"][i++]);
        }
        let j = 0;
        let myStr = "யாமறிந்த";
        for (const substr of myStr) {
            expect(substr).to.equal(["ய", "ா", "ம", "ற", "ி", "ந", "்", "த"][j++]);
        }
    });
    it("should support multiline strings as an object index", () => {
        const key = `str
			ing`;
        const obj = {
            [`str
			ing`]: "foo",
        };
        // prettier-ignore
        expect(obj[`str
			ing`]).to.equal("foo");
        expect(obj[key]).to.equal("foo");
        // prettier-ignore
        obj[`str
			ing`] = "bar";
        // prettier-ignore
        expect(obj[`str
			ing`]).to.equal("bar");
        expect(obj[key]).to.equal("bar");
    });
    // issue #1467
    it("should support strings with \", ', and ending with ]", () => {
        const str1 = `A string with a " and ' ending in a ]`;
        expect(str1.size()).to.equal(37);
        const str2 = `A string with a " and ' ending in a ]] ]=`;
        expect(str2.size()).to.equal(41);
    });
};
