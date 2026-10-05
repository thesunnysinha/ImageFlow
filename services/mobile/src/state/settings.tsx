import * as SecureStore from "expo-secure-store";
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { ApiClient } from "../api/client";

const BASE_URL_KEY = "imageflow.baseUrl";
const API_KEY_KEY = "imageflow.apiKey";

interface Settings {
  ready: boolean; // stored values have been read
  baseUrl: string;
  apiKey: string;
  client: ApiClient | null; // null until both values are set
  save: (baseUrl: string, apiKey: string) => Promise<void>;
  clear: () => Promise<void>;
}

const SettingsContext = createContext<Settings | null>(null);

/** Keeps the server address and API key in the platform keystore (Keychain / Keystore). */
export function SettingsProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [baseUrl, setBaseUrl] = useState("");
  const [apiKey, setApiKey] = useState("");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [url, key] = await Promise.all([SecureStore.getItemAsync(BASE_URL_KEY), SecureStore.getItemAsync(API_KEY_KEY)]);
        if (!cancelled) {
          setBaseUrl(url ?? "");
          setApiKey(key ?? "");
        }
      } finally {
        if (!cancelled) setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const save = useCallback(async (url: string, key: string) => {
    await Promise.all([SecureStore.setItemAsync(BASE_URL_KEY, url), SecureStore.setItemAsync(API_KEY_KEY, key)]);
    setBaseUrl(url);
    setApiKey(key);
  }, []);

  const clear = useCallback(async () => {
    await Promise.all([SecureStore.deleteItemAsync(BASE_URL_KEY), SecureStore.deleteItemAsync(API_KEY_KEY)]);
    setBaseUrl("");
    setApiKey("");
  }, []);

  const client = useMemo(() => (baseUrl && apiKey ? new ApiClient({ baseUrl, apiKey }) : null), [baseUrl, apiKey]);
  const value = useMemo(() => ({ ready, baseUrl, apiKey, client, save, clear }), [ready, baseUrl, apiKey, client, save, clear]);
  return <SettingsContext.Provider value={value}>{children}</SettingsContext.Provider>;
}

export function useSettings(): Settings {
  const value = useContext(SettingsContext);
  if (!value) throw new Error("useSettings must be used inside SettingsProvider");
  return value;
}
