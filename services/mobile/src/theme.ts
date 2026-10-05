import { useColorScheme } from "react-native";

const light = { bg: "#f6f7f9", card: "#ffffff", text: "#14181f", muted: "#5d6675", border: "#dfe3ea", primary: "#2457d6", onPrimary: "#ffffff", danger: "#c0392b", ok: "#1e8e4e", warn: "#b7791f" };
const dark = { bg: "#0f1218", card: "#1a1f29", text: "#eef1f6", muted: "#9aa4b5", border: "#2a3140", primary: "#6c93ff", onPrimary: "#0f1218", danger: "#ff7a6b", ok: "#4cc785", warn: "#e6b455" };

export type Theme = typeof light;

export function useTheme(): Theme {
  return useColorScheme() === "dark" ? dark : light;
}
