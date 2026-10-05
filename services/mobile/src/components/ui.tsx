import type { ReactNode } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Text, View, type StyleProp, type ViewStyle } from "react-native";
import type { ItemStatus, JobStatus } from "../api/types";
import { statusLabel } from "../lib/status";
import { useTheme } from "../theme";

export function Button({ title, onPress, disabled, busy, kind = "primary", style }: {
  title: string; onPress: () => void; disabled?: boolean; busy?: boolean; kind?: "primary" | "plain"; style?: StyleProp<ViewStyle>;
}) {
  const t = useTheme();
  const off = disabled || busy;
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: !!off, busy: !!busy }}
      onPress={onPress}
      disabled={off}
      style={({ pressed }) => [
        styles.button,
        kind === "primary" ? { backgroundColor: t.primary } : { borderColor: t.border, borderWidth: 1 },
        { opacity: off ? 0.5 : pressed ? 0.85 : 1 },
        style,
      ]}
    >
      {busy ? <ActivityIndicator color={kind === "primary" ? t.onPrimary : t.text} /> : (
        <Text style={[styles.buttonText, { color: kind === "primary" ? t.onPrimary : t.text }]}>{title}</Text>
      )}
    </Pressable>
  );
}

export function Card({ children, style }: { children: ReactNode; style?: StyleProp<ViewStyle> }) {
  const t = useTheme();
  return <View style={[styles.card, { backgroundColor: t.card, borderColor: t.border }, style]}>{children}</View>;
}

export function StatusBadge({ status }: { status: JobStatus | ItemStatus }) {
  const t = useTheme();
  const color = status === "completed" ? t.ok : status === "failed" ? t.danger : status === "partial" ? t.warn : t.muted;
  return (
    <View style={[styles.badge, { borderColor: color }]}>
      <Text style={{ color, fontSize: 12, fontWeight: "600" }}>{statusLabel[status]}</Text>
    </View>
  );
}

export function ProgressBar({ value }: { value: number }) {
  const t = useTheme();
  return (
    <View style={[styles.track, { backgroundColor: t.border }]} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: 100, now: Math.round(value * 100) }}>
      <View style={{ width: `${Math.round(value * 100)}%`, height: "100%", backgroundColor: t.primary }} />
    </View>
  );
}

export function Message({ text, kind = "muted" }: { text: string; kind?: "muted" | "error" }) {
  const t = useTheme();
  return <Text style={{ color: kind === "error" ? t.danger : t.muted, fontSize: 14 }}>{text}</Text>;
}

const styles = StyleSheet.create({
  button: { minHeight: 48, borderRadius: 12, alignItems: "center", justifyContent: "center", paddingHorizontal: 16 },
  buttonText: { fontSize: 16, fontWeight: "600" },
  card: { borderRadius: 14, borderWidth: StyleSheet.hairlineWidth, padding: 14, gap: 8 },
  badge: { borderWidth: 1, borderRadius: 999, paddingHorizontal: 10, paddingVertical: 2, alignSelf: "flex-start" },
  track: { height: 6, borderRadius: 3, overflow: "hidden" },
});
