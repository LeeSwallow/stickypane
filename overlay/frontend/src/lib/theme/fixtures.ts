// Mirror of internal/theme/palettes.go, for tests and the demo page only.
// At run time the palette comes from the bridge.
import type { Palette } from './palette'

export const THEMES: Palette[] = [
  {
    "name": "stickypane-dark",
    "dark": true,
    "base": "#1F2933",
    "text": "#E6E6E6",
    "muted": "#8A94A0",
    "accent": "#7FB3F5",
    "select": "#2F3B4C",
    "good": "#8FD694",
    "warn": "#F5A962",
    "bad": "#F28FB1",
    "info": "#B79CF2",
    "notes": [
      "#F2D45C",
      "#F28FB1",
      "#7FB3F5",
      "#8FD694",
      "#B79CF2",
      "#F5A962"
    ]
  },
  {
    "name": "stickypane-light",
    "dark": false,
    "base": "#FFFFFF",
    "text": "#1F2933",
    "muted": "#6B7684",
    "accent": "#2F6FD1",
    "select": "#D9E6FA",
    "good": "#2E8B57",
    "warn": "#C76B1A",
    "bad": "#C9407A",
    "info": "#7A4FC9",
    "notes": [
      "#B38F00",
      "#C9407A",
      "#2F6FD1",
      "#2E8B57",
      "#7A4FC9",
      "#C76B1A"
    ]
  },
  {
    "name": "catppuccin-mocha",
    "dark": true,
    "base": "#1e1e2e",
    "text": "#cdd6f4",
    "muted": "#9399b2",
    "accent": "#89b4fa",
    "select": "#45475a",
    "good": "#a6e3a1",
    "warn": "#f9e2af",
    "bad": "#f38ba8",
    "info": "#cba6f7",
    "notes": [
      "#f9e2af",
      "#f5c2e7",
      "#89b4fa",
      "#a6e3a1",
      "#cba6f7",
      "#fab387"
    ]
  },
  {
    "name": "catppuccin-latte",
    "dark": false,
    "base": "#eff1f5",
    "text": "#4c4f69",
    "muted": "#7c7f93",
    "accent": "#1e66f5",
    "select": "#ccd0da",
    "good": "#40a02b",
    "warn": "#df8e1d",
    "bad": "#d20f39",
    "info": "#8839ef",
    "notes": [
      "#df8e1d",
      "#ea76cb",
      "#1e66f5",
      "#40a02b",
      "#8839ef",
      "#fe640b"
    ]
  },
  {
    "name": "tokyonight-night",
    "dark": true,
    "base": "#1a1b26",
    "text": "#c0caf5",
    "muted": "#565f89",
    "accent": "#7aa2f7",
    "select": "#292e42",
    "good": "#9ece6a",
    "warn": "#e0af68",
    "bad": "#f7768e",
    "info": "#bb9af7",
    "notes": [
      "#e0af68",
      "#f7768e",
      "#7aa2f7",
      "#9ece6a",
      "#bb9af7",
      "#ff9e64"
    ]
  },
  {
    "name": "tokyonight-day",
    "dark": false,
    "base": "#e1e2e7",
    "text": "#3760bf",
    "muted": "#848cb5",
    "accent": "#2e7de9",
    "select": "#c4c8da",
    "good": "#587539",
    "warn": "#b15c00",
    "bad": "#f52a65",
    "info": "#9854f1",
    "notes": [
      "#8c6c3e",
      "#f52a65",
      "#2e7de9",
      "#587539",
      "#9854f1",
      "#b15c00"
    ]
  },
  {
    "name": "gruvbox-dark",
    "dark": true,
    "base": "#282828",
    "text": "#ebdbb2",
    "muted": "#928374",
    "accent": "#83a598",
    "select": "#504945",
    "good": "#b8bb26",
    "warn": "#fabd2f",
    "bad": "#fb4934",
    "info": "#d3869b",
    "notes": [
      "#fabd2f",
      "#d3869b",
      "#83a598",
      "#b8bb26",
      "#b16286",
      "#fe8019"
    ]
  },
  {
    "name": "gruvbox-light",
    "dark": false,
    "base": "#fbf1c7",
    "text": "#3c3836",
    "muted": "#928374",
    "accent": "#076678",
    "select": "#d5c4a1",
    "good": "#79740e",
    "warn": "#b57614",
    "bad": "#9d0006",
    "info": "#8f3f71",
    "notes": [
      "#b57614",
      "#8f3f71",
      "#076678",
      "#79740e",
      "#b16286",
      "#af3a03"
    ]
  },
  {
    "name": "nord",
    "dark": true,
    "base": "#2E3440",
    "text": "#D8DEE9",
    "muted": "#616E88",
    "accent": "#88C0D0",
    "select": "#434C5E",
    "good": "#A3BE8C",
    "warn": "#EBCB8B",
    "bad": "#BF616A",
    "info": "#B48EAD",
    "notes": [
      "#EBCB8B",
      "#BF616A",
      "#81A1C1",
      "#A3BE8C",
      "#B48EAD",
      "#D08770"
    ]
  },
  {
    "name": "dracula",
    "dark": true,
    "base": "#282A36",
    "text": "#F8F8F2",
    "muted": "#6272A4",
    "accent": "#BD93F9",
    "select": "#44475A",
    "good": "#50fa7b",
    "warn": "#F1FA8C",
    "bad": "#FF5555",
    "info": "#FF79C6",
    "notes": [
      "#F1FA8C",
      "#FF79C6",
      "#8BE9FD",
      "#50fa7b",
      "#BD93F9",
      "#FFB86C"
    ]
  }
]
