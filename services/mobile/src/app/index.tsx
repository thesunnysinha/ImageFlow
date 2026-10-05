import { useInfiniteQuery } from "@tanstack/react-query";
import { Link, router, Stack } from "expo-router";
import { ActivityIndicator, FlatList, Pressable, RefreshControl, StyleSheet, Text, View } from "react-native";
import { Button, Card, Message, ProgressBar, StatusBadge } from "../components/ui";
import { progress } from "../lib/status";
import { useSettings } from "../state/settings";
import { useTheme } from "../theme";

export default function JobsScreen() {
  const t = useTheme();
  const { client, ready } = useSettings();
  const jobs = useInfiniteQuery({
    queryKey: ["jobs", client?.outputUrl("", 0)], // changes when the server or key changes
    enabled: !!client,
    initialPageParam: null as string | null,
    queryFn: ({ pageParam }) => client!.listJobs({ limit: 20, before: pageParam }),
    getNextPageParam: (last) => last.nextBefore,
    // Keep the list fresh while something is still running.
    refetchInterval: (query) =>
      query.state.data?.pages.some((p) => p.items.some((j) => j.status === "queued" || j.status === "processing")) ? 4_000 : false,
  });

  const header = (
    <Stack.Screen options={{ headerRight: () => (
      <Link href="/settings" accessibilityLabel="Server settings" style={{ color: t.primary, fontSize: 16 }}>Server</Link>
    ) }} />
  );

  if (!ready) return <View style={styles.center}><ActivityIndicator /></View>;

  const deviceCard = (
    <Pressable accessibilityRole="button" accessibilityLabel="Compress photos on this device" onPress={() => router.push("/compress")}>
      <Card>
        <Text style={{ color: t.text, fontWeight: "700", fontSize: 16 }}>Compress photos on this device</Text>
        <Message text="Pick photos, shrink them, then save or share. Works offline; nothing is uploaded." />
      </Card>
    </Pressable>
  );

  if (!client) {
    return (
      <View style={styles.center}>
        {header}
        <View style={{ alignSelf: "stretch" }}>{deviceCard}</View>
        <Text style={[styles.title, { color: t.text, marginTop: 16 }]}>Batch jobs from URLs</Text>
        <Message text="Connect to your ImageFlow server to process lists of image links." />
        <Button title="Set up server" kind="plain" onPress={() => router.push("/settings")} style={{ alignSelf: "stretch" }} />
      </View>
    );
  }

  const items = jobs.data?.pages.flatMap((p) => p.items) ?? [];
  return (
    <View style={{ flex: 1 }}>
      {header}
      <FlatList
        contentContainerStyle={styles.list}
        data={items}
        keyExtractor={(j) => j.id}
        refreshControl={<RefreshControl refreshing={jobs.isRefetching && !jobs.isFetchingNextPage} onRefresh={() => jobs.refetch()} />}
        onEndReached={() => jobs.hasNextPage && !jobs.isFetchingNextPage && jobs.fetchNextPage()}
        onEndReachedThreshold={0.4}
        ListHeaderComponent={<View style={{ marginBottom: 12 }}>{deviceCard}</View>}
        ListEmptyComponent={
          jobs.isPending ? <ActivityIndicator style={{ marginTop: 40 }} /> :
          jobs.isError ? <View style={{ gap: 12, marginTop: 24 }}><Message kind="error" text={jobs.error.message} /><Button title="Try again" kind="plain" onPress={() => jobs.refetch()} /></View> :
          <Message text="No jobs yet. Tap “New job” to compress your first images." />
        }
        ListFooterComponent={jobs.isFetchingNextPage ? <ActivityIndicator style={{ margin: 16 }} /> : null}
        renderItem={({ item }) => (
          <Pressable accessibilityRole="button" onPress={() => router.push({ pathname: "/job/[id]", params: { id: item.id } })}>
            <Card>
              <View style={styles.row}>
                <Text style={{ color: t.text, fontWeight: "600" }}>{item.counts.total} image{item.counts.total === 1 ? "" : "s"}</Text>
                <StatusBadge status={item.status} />
              </View>
              <ProgressBar value={progress(item.counts)} />
              <Text style={{ color: t.muted, fontSize: 12 }}>{new Date(item.created_at).toLocaleString()}</Text>
            </Card>
          </Pressable>
        )}
        ItemSeparatorComponent={() => <View style={{ height: 10 }} />}
      />
      <View style={[styles.footer, { backgroundColor: t.bg, borderColor: t.border }]}>
        <Button title="New job" onPress={() => router.push("/new")} />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24, gap: 12 },
  title: { fontSize: 20, fontWeight: "700" },
  list: { padding: 16, paddingBottom: 24 },
  row: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  footer: { padding: 16, borderTopWidth: StyleSheet.hairlineWidth },
});
