import { useQuery } from "@tanstack/react-query";
import { Stack, useLocalSearchParams } from "expo-router";
import { ActivityIndicator, FlatList, Image, StyleSheet, Text, View } from "react-native";
import type { Item } from "../../api/types";
import { Button, Card, Message, ProgressBar, StatusBadge } from "../../components/ui";
import { formatBytes, isTerminal, progress, savings } from "../../lib/status";
import { useSettings } from "../../state/settings";
import { useTheme } from "../../theme";

export default function JobScreen() {
  const t = useTheme();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { client } = useSettings();
  const job = useQuery({
    queryKey: ["job", id],
    enabled: !!client && !!id,
    queryFn: () => client!.getJob(id),
    // Poll every 2 s until the job reaches a final state.
    refetchInterval: (query) => (query.state.data && isTerminal(query.state.data.status) ? false : 2_000),
  });

  if (job.isPending) return <View style={styles.center}><ActivityIndicator /></View>;
  if (job.isError) {
    return (
      <View style={styles.center}>
        <Message kind="error" text={job.error.message} />
        <Button title="Try again" kind="plain" onPress={() => job.refetch()} />
      </View>
    );
  }
  const data = job.data;
  const counts = {
    total: data.items.length,
    completed: data.items.filter((i) => i.status === "completed").length,
    failed: data.items.filter((i) => i.status === "failed").length,
  };
  return (
    <FlatList
      contentContainerStyle={styles.list}
      data={data.items}
      keyExtractor={(i) => i.id}
      ItemSeparatorComponent={() => <View style={{ height: 10 }} />}
      ListHeaderComponent={
        <Card style={{ marginBottom: 12 }}>
          <Stack.Screen options={{ title: `Job ${data.id.slice(0, 8)}` }} />
          <View style={styles.row}>
            <Text style={{ color: t.text, fontWeight: "700", fontSize: 16 }}>{counts.completed} of {counts.total} done</Text>
            <StatusBadge status={data.status} />
          </View>
          <ProgressBar value={progress(counts)} />
          {counts.failed > 0 && <Text style={{ color: t.danger, fontSize: 13 }}>{counts.failed} failed</Text>}
        </Card>
      }
      renderItem={({ item }) => <ItemRow item={item} jobId={data.id} />}
    />
  );
}

function ItemRow({ item, jobId }: { item: Item; jobId: string }) {
  const t = useTheme();
  const { client } = useSettings();
  const saved = savings(item.bytes_in, item.bytes_out);
  return (
    <Card>
      <View style={styles.row}>
        <Text style={{ color: t.text, flex: 1, marginRight: 8 }} numberOfLines={1}>{item.source_url}</Text>
        <StatusBadge status={item.status} />
      </View>
      {item.status === "completed" && client && (
        <>
          {/* The output is protected, so the image request carries the API key. */}
          <Image
            source={{ uri: client.outputUrl(jobId, item.position), headers: client.authHeaders }}
            style={[styles.thumb, { backgroundColor: t.border }]} resizeMode="contain"
            accessibilityLabel={`Compressed image ${item.position + 1}`}
          />
          <Text style={{ color: t.muted, fontSize: 13 }}>
            {formatBytes(item.bytes_in)} → {formatBytes(item.bytes_out)} {saved && `(${saved})`}
          </Text>
        </>
      )}
      {item.status === "failed" && <Text style={{ color: t.danger, fontSize: 13 }}>{item.error ?? "Failed"}</Text>}
    </Card>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24, gap: 12 },
  list: { padding: 16 },
  row: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  thumb: { width: "100%", height: 200, borderRadius: 10 },
});
