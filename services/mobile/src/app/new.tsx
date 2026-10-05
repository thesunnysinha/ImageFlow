import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router } from "expo-router";
import { useMemo, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput } from "react-native";
import { ApiError } from "../api/client";
import { Button, Message } from "../components/ui";
import { isHttpUrl, parseUrlList } from "../lib/urls";
import { useSettings } from "../state/settings";
import { useTheme } from "../theme";

export default function NewJobScreen() {
  const t = useTheme();
  const { client } = useSettings();
  const queryClient = useQueryClient();
  const [text, setText] = useState("");
  const [webhook, setWebhook] = useState("");
  const parsed = useMemo(() => parseUrlList(text), [text]);
  const webhookBad = webhook.trim() !== "" && !isHttpUrl(webhook.trim());

  const create = useMutation({
    mutationFn: () => client!.createJob({ sourceUrls: parsed.urls, webhookUrl: webhook.trim() || undefined }),
    onSuccess: (job) => {
      void queryClient.invalidateQueries({ queryKey: ["jobs"] });
      router.replace({ pathname: "/job/[id]", params: { id: job.id } });
    },
  });

  const inputStyle = [styles.input, { color: t.text, backgroundColor: t.card, borderColor: t.border }];
  const error = create.error;
  const serverMessage = error instanceof ApiError && error.details.length > 0
    ? error.details.map((d) => `${d.field}: ${d.message}`).join("\n")
    : error?.message;

  return (
    <KeyboardAvoidingView style={{ flex: 1 }} behavior={Platform.OS === "ios" ? "padding" : undefined}>
      <ScrollView contentContainerStyle={styles.body} keyboardShouldPersistTaps="handled">
        <Text style={[styles.label, { color: t.text }]}>Image URLs</Text>
        <TextInput
          style={[...inputStyle, styles.multiline]} multiline value={text} onChangeText={setText}
          placeholder={"https://example.com/photo1.jpg\nhttps://example.com/photo2.png"} placeholderTextColor={t.muted}
          autoCapitalize="none" autoCorrect={false} keyboardType="url" accessibilityLabel="Image URLs"
        />
        <Message text={`${parsed.urls.length} valid URL${parsed.urls.length === 1 ? "" : "s"}${parsed.invalid.length ? `, ${parsed.invalid.length} ignored` : ""}. One per line, or separated by commas.`} />
        {parsed.invalid.length > 0 && <Message kind="error" text={`Not valid http(s) URLs: ${parsed.invalid.slice(0, 3).join(", ")}${parsed.invalid.length > 3 ? "…" : ""}`} />}

        <Text style={[styles.label, { color: t.text }]}>Webhook (optional)</Text>
        <TextInput
          style={inputStyle} value={webhook} onChangeText={setWebhook} placeholder="https://example.com/hooks/imageflow"
          placeholderTextColor={t.muted} autoCapitalize="none" autoCorrect={false} keyboardType="url" accessibilityLabel="Webhook URL"
        />
        {webhookBad && <Message kind="error" text="The webhook must be an http(s) URL." />}

        {error && <Message kind="error" text={serverMessage ?? "Something went wrong."} />}
        <Button title="Start compressing" onPress={() => create.mutate()} busy={create.isPending} disabled={!client || parsed.urls.length === 0 || webhookBad} />
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  body: { padding: 16, gap: 10 },
  label: { fontSize: 15, fontWeight: "600", marginTop: 6 },
  input: { borderWidth: 1, borderRadius: 12, paddingHorizontal: 12, paddingVertical: 10, fontSize: 15 },
  multiline: { minHeight: 140, textAlignVertical: "top" },
});
