import * as ImagePicker from "expo-image-picker";
import * as Sharing from "expo-sharing";
import { useRef, useState } from "react";
import { Alert, FlatList, Image, Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import { Button, Card, Message } from "../components/ui";
import { MAX_PHOTOS, QUALITY_PRESETS, SIZE_PRESETS } from "../lib/compress-core";
import { compressOnDevice, type DeviceResult } from "../lib/device-compress";
import { formatBytes, savings } from "../lib/status";
import { useTheme } from "../theme";

interface Photo {
  key: string;
  uri: string;
  width: number;
  height: number;
  name: string;
  bytes?: number;
  status: "ready" | "working" | "done" | "error";
  result?: DeviceResult;
  error?: string;
}

export default function CompressScreen() {
  const t = useTheme();
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [quality, setQuality] = useState<number>(QUALITY_PRESETS[1].quality);
  const [maxDimension, setMaxDimension] = useState<number | null>(null);
  const [targetKB, setTargetKB] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");
  const counter = useRef(0);

  async function pick() {
    const room = MAX_PHOTOS - photos.length;
    if (room <= 0) {
      setNotice(`Up to ${MAX_PHOTOS} photos at a time.`);
      return;
    }
    const picked = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"], allowsMultipleSelection: true, selectionLimit: room, quality: 1, exif: false,
    });
    if (picked.canceled) return;
    setNotice("");
    setPhotos((cur) => [
      ...cur,
      ...picked.assets.slice(0, room).map((a): Photo => ({
        key: `p${counter.current++}`, uri: a.uri, width: a.width, height: a.height,
        name: a.fileName ?? "photo.jpg", bytes: a.fileSize, status: "ready",
      })),
    ]);
  }

  async function run() {
    setBusy(true);
    setNotice("");
    const kb = Number.parseInt(targetKB, 10);
    const options = { quality, maxDimension, targetBytes: Number.isFinite(kb) && kb >= 10 ? kb * 1024 : null };
    // One photo at a time: decoding several 12-megapixel images together can exhaust a phone's memory.
    for (const photo of photos) {
      setPhotos((cur) => cur.map((p) => (p.key === photo.key ? { ...p, status: "working", error: undefined } : p)));
      try {
        const result = await compressOnDevice(photo, options);
        setPhotos((cur) => cur.map((p) => (p.key === photo.key ? { ...p, status: "done", result } : p)));
      } catch (e) {
        setPhotos((cur) => cur.map((p) => (p.key === photo.key ? { ...p, status: "error", error: e instanceof Error ? e.message : "Failed" } : p)));
      }
    }
    setBusy(false);
  }

  async function share(photo: Photo) {
    if (!photo.result) return;
    if (!(await Sharing.isAvailableAsync())) {
      Alert.alert("Sharing is not available on this device.");
      return;
    }
    await Sharing.shareAsync(photo.result.uri, { mimeType: "image/jpeg", dialogTitle: "Save or share" });
  }

  const done = photos.filter((p) => p.result);
  const before = done.reduce((n, p) => n + (p.result?.originalBytes ?? 0), 0);
  const after = done.reduce((n, p) => n + (p.result?.bytes ?? 0), 0);
  const chip = (selected: boolean) => [styles.chip, { borderColor: selected ? t.primary : t.border, backgroundColor: selected ? t.primary : t.card }];
  const chipText = (selected: boolean) => ({ color: selected ? t.onPrimary : t.text, fontWeight: "600" as const, fontSize: 14 });

  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={photos}
      keyExtractor={(p) => p.key}
      keyboardShouldPersistTaps="handled"
      ItemSeparatorComponent={() => <View style={{ height: 10 }} />}
      ListHeaderComponent={
        <View style={{ gap: 12, marginBottom: 12 }}>
          <Message text="Photos are compressed on your phone. Nothing is uploaded." />
          <Button title={photos.length ? "Add more photos" : "Choose photos"} onPress={pick} disabled={busy} />

          <Text style={[styles.label, { color: t.text }]}>Quality</Text>
          <View style={styles.chips}>
            {QUALITY_PRESETS.map((q) => (
              <Pressable key={q.id} accessibilityRole="button" accessibilityState={{ selected: quality === q.quality }}
                style={chip(quality === q.quality)} onPress={() => setQuality(q.quality)}>
                <Text style={chipText(quality === q.quality)}>{q.label}</Text>
              </Pressable>
            ))}
          </View>

          <Text style={[styles.label, { color: t.text }]}>Longest side (px)</Text>
          <View style={styles.chips}>
            {SIZE_PRESETS.map((s) => (
              <Pressable key={s.label} accessibilityRole="button" accessibilityState={{ selected: maxDimension === s.maxDimension }}
                style={chip(maxDimension === s.maxDimension)} onPress={() => setMaxDimension(s.maxDimension)}>
                <Text style={chipText(maxDimension === s.maxDimension)}>{s.label}</Text>
              </Pressable>
            ))}
          </View>

          <Text style={[styles.label, { color: t.text }]}>Maximum size (KB, optional)</Text>
          <TextInput style={[styles.input, { color: t.text, backgroundColor: t.card, borderColor: t.border }]} value={targetKB}
            onChangeText={(v) => setTargetKB(v.replace(/[^0-9]/g, ""))} placeholder="e.g. 200" placeholderTextColor={t.muted}
            keyboardType="number-pad" accessibilityLabel="Maximum size in kilobytes" />

          {notice !== "" && <Message kind="error" text={notice} />}
          {photos.length > 0 && (
            <View style={{ flexDirection: "row", gap: 8 }}>
              <Button title={busy ? "Compressing…" : done.length ? "Compress again" : "Compress"} onPress={run} busy={busy} style={{ flex: 1 }} />
              <Button title="Clear" kind="plain" disabled={busy} onPress={() => setPhotos([])} />
            </View>
          )}
          {done.length > 0 && (
            <Text style={{ color: t.text, fontSize: 16 }}>
              {formatBytes(before)} → <Text style={{ fontWeight: "700" }}>{formatBytes(after)}</Text> {savings(before, after) && `(${savings(before, after)})`}
            </Text>
          )}
        </View>
      }
      renderItem={({ item }) => (
        <Card>
          <View style={styles.row}>
            <Image source={{ uri: item.result?.uri ?? item.uri }} style={[styles.thumb, { backgroundColor: t.border }]} accessibilityLabel={`Preview of ${item.name}`} />
            <View style={{ flex: 1, gap: 2 }}>
              <Text style={{ color: t.text, fontWeight: "600" }} numberOfLines={1}>{item.name}</Text>
              <Text style={{ color: t.muted, fontSize: 13 }}>{item.width} × {item.height}{item.bytes ? ` · ${formatBytes(item.bytes)}` : ""}</Text>
              {item.status === "working" && <Text style={{ color: t.muted, fontSize: 13 }}>Working…</Text>}
              {item.status === "error" && <Text style={{ color: t.danger, fontSize: 13 }}>{item.error}</Text>}
              {item.result && (
                <Text style={{ color: t.text, fontSize: 13 }}>
                  → <Text style={{ fontWeight: "700" }}>{formatBytes(item.result.bytes)}</Text> {savings(item.result.originalBytes, item.result.bytes)}
                  {item.result.keptOriginal ? " (already small, kept as is)" : ""}
                  {!item.result.reachedTarget ? " (smallest possible; try a smaller side)" : ""}
                </Text>
              )}
            </View>
          </View>
          {item.result && <Button title="Save or share" kind="plain" onPress={() => void share(item)} />}
        </Card>
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: 16, paddingBottom: 32 },
  label: { fontSize: 15, fontWeight: "600", marginTop: 4 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  chip: { borderWidth: 1, borderRadius: 999, paddingHorizontal: 14, minHeight: 40, alignItems: "center", justifyContent: "center" },
  input: { borderWidth: 1, borderRadius: 12, paddingHorizontal: 12, paddingVertical: 10, fontSize: 15 },
  row: { flexDirection: "row", gap: 12, alignItems: "center" },
  thumb: { width: 64, height: 64, borderRadius: 8 },
});
