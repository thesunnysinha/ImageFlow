import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Stack } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { useState } from "react";
import { SettingsProvider } from "../state/settings";
import { useTheme } from "../theme";

export default function RootLayout() {
  const t = useTheme();
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: 1, staleTime: 5_000 } } }));
  return (
    <QueryClientProvider client={queryClient}>
      <SettingsProvider>
        <StatusBar style="auto" />
        <Stack screenOptions={{ headerStyle: { backgroundColor: t.card }, headerTintColor: t.text, contentStyle: { backgroundColor: t.bg } }}>
          <Stack.Screen name="index" options={{ title: "ImageFlow" }} />
          <Stack.Screen name="compress" options={{ title: "Compress photos" }} />
          <Stack.Screen name="new" options={{ title: "New job", presentation: "modal" }} />
          <Stack.Screen name="settings" options={{ title: "Server", presentation: "modal" }} />
          <Stack.Screen name="job/[id]" options={{ title: "Job" }} />
        </Stack>
      </SettingsProvider>
    </QueryClientProvider>
  );
}
