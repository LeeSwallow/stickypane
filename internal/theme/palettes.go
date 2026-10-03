package theme

// all holds the built-in themes. The stickypane ones are the colors the
// board started with; the others are the palettes of well-known editor
// themes, with every value taken from the theme's source as it is. The
// roles are filled the way those themes fill their own: the comment color
// for what is muted, a surface grey for the selection, blue for focus,
// mauve or purple for headings, and red, green and yellow for state.
var all = []Theme{
	{
		Name: DefaultDark, Base: "#1F2933", Dark: true, Source: "stickypane's own palette",
		Text: "#E6E6E6", Muted: "#8A94A0", Accent: "#7FB3F5", Select: "#2F3B4C",
		Good: "#8FD694", Warn: "#F5A962", Bad: "#F28FB1", Info: "#B79CF2",
		Notes: [6]string{"#F2D45C", "#F28FB1", "#7FB3F5", "#8FD694", "#B79CF2", "#F5A962"},
	},
	{
		Name: DefaultLight, Base: "#FFFFFF", Dark: false, Source: "stickypane's own palette",
		Text: "#1F2933", Muted: "#6B7684", Accent: "#2F6FD1", Select: "#D9E6FA",
		Good: "#2E8B57", Warn: "#C76B1A", Bad: "#C9407A", Info: "#7A4FC9",
		Notes: [6]string{"#B38F00", "#C9407A", "#2F6FD1", "#2E8B57", "#7A4FC9", "#C76B1A"},
	},
	{
		Name: "catppuccin-mocha", Base: "#1e1e2e", Dark: true, Source: "catppuccin/nvim, lua/catppuccin/palettes/mocha.lua (MIT)",
		Text: "#cdd6f4", Muted: "#9399b2", Accent: "#89b4fa", Select: "#45475a",
		Good: "#a6e3a1", Warn: "#f9e2af", Bad: "#f38ba8", Info: "#cba6f7",
		Notes: [6]string{"#f9e2af", "#f5c2e7", "#89b4fa", "#a6e3a1", "#cba6f7", "#fab387"},
	},
	{
		Name: "catppuccin-latte", Base: "#eff1f5", Dark: false, Source: "catppuccin/nvim, lua/catppuccin/palettes/latte.lua (MIT)",
		Text: "#4c4f69", Muted: "#7c7f93", Accent: "#1e66f5", Select: "#ccd0da",
		Good: "#40a02b", Warn: "#df8e1d", Bad: "#d20f39", Info: "#8839ef",
		Notes: [6]string{"#df8e1d", "#ea76cb", "#1e66f5", "#40a02b", "#8839ef", "#fe640b"},
	},
	{
		Name: "tokyonight-night", Base: "#1a1b26", Dark: true, Source: "folke/tokyonight.nvim, lua/tokyonight/colors/{storm,night}.lua (Apache-2.0)",
		Text: "#c0caf5", Muted: "#565f89", Accent: "#7aa2f7", Select: "#292e42",
		Good: "#9ece6a", Warn: "#e0af68", Bad: "#f7768e", Info: "#bb9af7",
		Notes: [6]string{"#e0af68", "#f7768e", "#7aa2f7", "#9ece6a", "#bb9af7", "#ff9e64"},
	},
	{
		Name: "tokyonight-day", Base: "#e1e2e7", Dark: false, Source: "folke/tokyonight.nvim, extras/lua/tokyonight_day.lua (Apache-2.0)",
		Text: "#3760bf", Muted: "#848cb5", Accent: "#2e7de9", Select: "#c4c8da",
		Good: "#587539", Warn: "#b15c00", Bad: "#f52a65", Info: "#9854f1",
		Notes: [6]string{"#8c6c3e", "#f52a65", "#2e7de9", "#587539", "#9854f1", "#b15c00"},
	},
	{
		Name: "gruvbox-dark", Base: "#282828", Dark: true, Source: "ellisonleao/gruvbox.nvim, lua/gruvbox.lua (MIT)",
		Text: "#ebdbb2", Muted: "#928374", Accent: "#83a598", Select: "#504945",
		Good: "#b8bb26", Warn: "#fabd2f", Bad: "#fb4934", Info: "#d3869b",
		Notes: [6]string{"#fabd2f", "#d3869b", "#83a598", "#b8bb26", "#b16286", "#fe8019"},
	},
	{
		Name: "gruvbox-light", Base: "#fbf1c7", Dark: false, Source: "ellisonleao/gruvbox.nvim, lua/gruvbox.lua (MIT)",
		Text: "#3c3836", Muted: "#928374", Accent: "#076678", Select: "#d5c4a1",
		Good: "#79740e", Warn: "#b57614", Bad: "#9d0006", Info: "#8f3f71",
		Notes: [6]string{"#b57614", "#8f3f71", "#076678", "#79740e", "#b16286", "#af3a03"},
	},
	{
		Name: "nord", Base: "#2E3440", Dark: true, Source: "gbprod/nord.nvim, lua/nord/colors.lua (WTFPL)",
		Text: "#D8DEE9", Muted: "#616E88", Accent: "#88C0D0", Select: "#434C5E",
		Good: "#A3BE8C", Warn: "#EBCB8B", Bad: "#BF616A", Info: "#B48EAD",
		Notes: [6]string{"#EBCB8B", "#BF616A", "#81A1C1", "#A3BE8C", "#B48EAD", "#D08770"},
	},
	{
		Name: "dracula", Base: "#282A36", Dark: true, Source: "Mofiqul/dracula.nvim, lua/dracula/palette.lua (MIT)",
		Text: "#F8F8F2", Muted: "#6272A4", Accent: "#BD93F9", Select: "#44475A",
		Good: "#50fa7b", Warn: "#F1FA8C", Bad: "#FF5555", Info: "#FF79C6",
		Notes: [6]string{"#F1FA8C", "#FF79C6", "#8BE9FD", "#50fa7b", "#BD93F9", "#FFB86C"},
	},
}
