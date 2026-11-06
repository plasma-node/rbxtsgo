// The string matrix: one file per escape case, and what JS says each one's value is.
// Run: node gen.cjs <project folder>. Writes <project folder>/src/*.ts and <project folder>/expected.json.
// In the cases, ~ is a backslash and <LF> <CR> <LS> <PS> are raw characters, so that this file stays plain text.
"use strict";
const fs = require("fs");

const cases = {
	// a backslash and a line break in a plain string
	s_lf: '"Hello~<LF>World!"',
	s_crlf: '"Hello~<CR><LF>World!"',
	s_cr: '"Hello~<CR>World!"',
	s_ls: '"Hello~<LS>World!"',
	s_ps: '"Hello~<PS>World!"',
	sq_lf: "'Hello~<LF>World!'",
	s_lf_quote: '"He~"llo~<LF>World!"',
	s_lf_u: '"~u0041~<LF>b"',
	// a backslash and a line break in a template
	t_lf: "`Hello~<LF>World!`",
	t_crlf: "`Hello~<CR><LF>World!`",
	t_cr: "`Hello~<CR>World!`",
	t_ls: "`Hello~<LS>World!`",
	t_ps: "`Hello~<PS>World!`",
	ti_lf: "`Hello~<LF>World ${x}!`",
	ti_crlf: "`Hello~<CR><LF>World ${x}!`",
	ti_ls: "`Hello~<LS>World ${x}!`",
	ti_tail: "`${x}Hello~<LF>World`",
	ti_middle: "`${x}a~<LF>b${x}`",
	bslash_lf_ti: "`~~~<LF> ${x}`",
	// line breaks in a template's source
	t_raw_lf: "`line1<LF>line2`",
	t_raw_crlf: "`line1<CR><LF>line2`",
	t_raw_cr: "`line1<CR>line2`",
	ti_raw_crlf: "`line1<CR><LF>line2 ${x}`",
	ti_raw_lf: "`line1<LF>line2 ${x}`",
	// Unicode escapes
	u_s: '"~u0041"',
	u_t: "`~u0041`",
	u_ti: "`~u0041 ${x}`",
	u_lead: '"~u00e9x"',
	ub_s: '"~u{41}"',
	ub_ti: "`~u{41} ${x}`",
	u_pair: '"~uD83D~uDE00"',
	u_pair_lower: '"~ud83d~ude00"',
	u_pair_t: "`~uD83D~uDE00 ${x}`",
	u_bs: '"~~u0041"',
	u_bs_t: "`~~u0041 ${x}`",
	ub_bs_t: "`~~u{41}`",
	ub_bs_ti: "`~~u{41} ${x}`",
	u_long: "\"~\"'~u0041\"",
	// identity escapes
	id_a: '"~a"',
	id_z: '"a~z   b"',
	id_q: '"~q~d~/"',
	id_ti: "`~a~q ${x}`",
	// quotes
	q_mix: '"a~"b\'c"',
	q_mix2: "'a~'b\"c'",
	q_mix_n: '"a~"b\'~nc"',
	q_plain: '"a\'b"',
	q_esc: '"a~\'b"',
	q_esc2: "'a~\"b'",
	// everything else that has to stay
	x: '"~x41"',
	bs: '"a~~b"',
	nul: '"a~0b"',
	tick_ti: "`a~`b ${x}`",
	tick_t: "`a~`b`",
	tick_s: '"a~`b"',
	dollar_t: "`a~${b}`",
	dollar_ti: "`a~${b} ${x}`",
	brace_ti: "`{} ${x}`",
	brace_t: "`{}`",
	esc_brace_ti: "`~{ ${x}`",
	esc_brace_t: "`~{`",
	esc_rbrace_t: "`~}`",
	esc_rbrace_ti: "`~} ${x}`",
	esc_both_t: "`~{a~}`",
	bslash_ti: "`~~ ${x}`",
	bslash_t: "`~~`",
	bslash_brace_ti: "`~~{ ${x}`",
	dq_ti: "`\"' ${x}`",
	nl_esc_ti: "`a~nb ${x}`",
	nl_esc_t: "`a~nb`",
	ctrl_s: '"a~tb~r~n~v~f~b"',
	plain: '"plain"',
	empty: '""',
	empty_t: "``",
	non_ascii: '"héllo 😀"',
	non_ascii_t: "`héllo 😀 ${x}`",
};

const raw = text =>
	text
		.replaceAll("~", String.fromCharCode(92))
		.replaceAll("<LF>", "\n")
		.replaceAll("<CR>", "\r")
		.replaceAll("<LS>", " ")
		.replaceAll("<PS>", " ");

const dir = process.argv[2];
fs.mkdirSync(dir + "/src", { recursive: true });
const expected = {};
for (const name of Object.keys(cases)) {
	const source = raw(cases[name]);
	fs.writeFileSync(`${dir}/src/${name}.ts`, `const x = 1;\nprint(x);\nexport = ${source};\n`);
	const x = 1;
	const b = undefined;
	expected[name] = Buffer.from(eval(source), "utf8").toString("hex");
}
fs.writeFileSync(dir + "/expected.json", JSON.stringify(expected));
