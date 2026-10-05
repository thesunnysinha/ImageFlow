import { useQueryClient } from "@tanstack/react-query";
import { router } from "expo-router";
import { useState } from "react";
import { Alert, ScrollView, StyleSheet, Text, TextInput } from "react-native";
import { ApiClient } from "../api/client";
import { Button, Message } from "../components/ui";
import { normalizeBaseUrl } from "../lib/urls";
import { useSettings } from "../state/settings";
import { useTheme } from "../theme";

export default function SettingsScreen() {
  const t = useTheme();
  const settings = useSettings();
  const queryClient = useQueryClient();
  const [baseUrl, setBaseUrl] = useState(settings.baseUrl);
  const [apiKey, setApiKey] = useState(settings.apiKey);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const normalized = normalizeBaseUrl(baseUrl);

  // Check the address and key with a real request before saving them.
  async function saveAndTest() {
    if (!normalized || !apiKey.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await new ApiClient({ baseUrl: normalized, apiKey: apiKey.trim() }).listJobs({ limit: 1 });
      await settings.save(normalized, apiKey.trim());
      await queryClient.invalidateQueries({ queryKey: ["jobs"] });
      router.back();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not connect.");
    } finally {
      setBusy(false);
    }
  }

  function signOut() {
    Alert.alert("Remove server?", "The address and API key are deleted from this device.", [
      { text: "Cancel", style: "cancel" },
      { text: "Remove", style: "destructive", onPress: () => { void settings.clear().then(() => { queryClient.clear(); router.back(); }); } },
    ]);
  }

  const inputStyle = [styles.input, { color: t.text, backgroundColor: t.card, borderColor: t.border }];
  return (
    <ScrollView contentContainerStyle={styles.body} keyboardShouldPersistTaps="handled">
      <Text style={[styles.label, { color: t.text }]}>Server address</Text>
      <TextInput style={inputStyle} value={baseUrl} onChangeText={setBaseUrl} placeholder="https://api.example.com" placeholderTextColor={t.muted}
        autoCapitalize="none" autoCorrect={false} keyboardType="url" accessibilityLabel="Server address" />
      {baseUrl.trim() !== "" && !normalized && <Message kind="error" text="Enter a valid http(s) address." />}
      <Text style={[styles.label, { color: t.text }]}>API key</Text>
      <TextInput style={inputStyle} value={apiKey} onChangeText={setApiKey} placeholder="Your API key" placeholderTextColor={t.muted}
        autoCapitalize="none" autoCorrect={false} secureTextEntry accessibilityLabel="API key" />
      <Message text="Stored in the device keystore. Use https for any server outside your own network." />
      {error && <Message kind="error" text={error} />}
      <Button title="Test and save" onPress={saveAndTest} busy={busy} disabled={!normalized || !apiKey.trim()} />
      {settings.client && <Button title="Remove server" kind="plain" onPress={signOut} />}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  body: { padding: 16, gap: 10 },
  label: { fontSize: 15, fontWeight: "600", marginTop: 6 },
  input: { borderWidth: 1, borderRadius: 12, paddingHorizontal: 12, paddingVertical: 10, fontSize: 15 },
});
