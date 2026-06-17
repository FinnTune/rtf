const js = require("@eslint/js");
const globals = require("globals");

module.exports = [
	js.configs.recommended,
	{
		files: ["frontend/**/*.js"],
		languageOptions: {
			ecmaVersion: "latest",
			sourceType: "module",
			globals: globals.browser,
		},
	},
];
