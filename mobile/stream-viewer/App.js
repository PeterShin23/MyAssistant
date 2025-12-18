import React, { useState, useEffect, useRef } from "react";
import {
  Dimensions,
  View,
  TextInput,
  Button,
  StyleSheet,
  ScrollView,
  Text,
  Platform,
  TouchableOpacity,
} from "react-native";
import Icon from "react-native-vector-icons/MaterialIcons";
import MDViewer from "./MDViewer";

const screen = Dimensions.get("screen");

const address = "";

export default function App() {
  const [wsUrl, setWsUrl] = useState(`ws://${address}:4000/stream?role=viewer`);
  const [isConnected, setIsConnected] = useState(false);
  const [content, setContent] = useState("");
  const [isLandscape, setIsLandscape] = useState(screen.width > screen.height);

  const scrollRef = useRef(null);
  const userHoldingRef = useRef(false);
  const wsRef = useRef(null);

  // Simple coalescing buffer to reduce re-renders
  const pendingRef = useRef("");
  const rafRef = useRef(null);
  const flushScheduledRef = useRef(false); // guard to avoid rescheduling within same frame

  Dimensions.addEventListener("change", ({ screen }) => {
    if (isLandscape !== screen.width > screen.height) {
      setIsLandscape(screen.width > screen.height ? true : false);
    }
  });

  useEffect(() => {
    // Clean up WebSocket connection on unmount
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      // Final flush so nothing is stranded
      if (pendingRef.current.length) {
        setContent((prev) => prev + pendingRef.current);
        pendingRef.current = "";
      }
      rafRef.current = null;
      flushScheduledRef.current = false;
    };
  }, []);

  const flush = () => {
    // Move pending into state once per frame
    const delta = pendingRef.current;
    if (delta.length) {
      setContent((prev) => prev + delta); // functional update avoids stale closure
      pendingRef.current = "";
    }
    rafRef.current = null;
    flushScheduledRef.current = false;

    // // Auto-scroll if user hasn't “held” the view
    // if (!userHoldingRef.current) {
    //   requestAnimationFrame(() => scrollRef.current?.scrollToEnd({ animated: true }));
    // }
  };

  const scheduleFlush = () => {
    // Schedule exactly one flush per frame; do not cancel an existing RAF
    if (flushScheduledRef.current) return;
    flushScheduledRef.current = true;
    if (!rafRef.current) {
      rafRef.current = requestAnimationFrame(flush);
    }
  };

  const connect = () => {
    // Close existing connection if any
    if (wsRef.current) {
      wsRef.current.close();
    }

    // Create new WebSocket connection
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      // console.log('[Frontend] Received WebSocket message:', event.data);

      let chunk = "";
      try {
        // Parse the JSON message
        const data =
          typeof event.data === "string" ? event.data : String(event.data);

        const parsed = JSON.parse(data);

        chunk = typeof parsed?.chunk === "string" ? parsed.chunk : data; // fall back to raw
        // console.log('[Frontend] Extracted chunk:', chunk);
      } catch (error) {
        // console.log('[Frontend] JSON parse failed, using raw data:', error.message);
        // not JSON? append raw
        chunk =
          typeof event.data === "string" ? event.data : String(event.data);
      }

      // console.log('[Frontend] Final chunk to append:', chunk);
      // console.log('[Frontend] Pending content before append:', pendingRef.current.length, 'chars');

      pendingRef.current += chunk;
      // console.log('[Frontend] Pending content after append:', pendingRef.current.length, 'chars');

      // Schedule one flush per frame (do NOT cancel an existing RAF)
      scheduleFlush();
    };

    ws.onclose = () => {
      console.log("[Frontend] WebSocket connection closed");
      // Process any remaining chunks before disconnecting
      if (pendingRef.current.length > 0) {
        // console.log('[Frontend] Processing remaining chunks before close:', pendingRef.current.length);
        setContent((prev) => prev + pendingRef.current);
        pendingRef.current = "";
      }
      rafRef.current = null;
      flushScheduledRef.current = false;
      setIsConnected(false);
    };

    ws.onerror = (error) => {
      console.log("WebSocket error:", error);
    };
  };

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    setIsConnected(false);
  };

  const clearContent = () => {
    setContent("");
    pendingRef.current = "";
    if (rafRef.current) {
      cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
    }
    flushScheduledRef.current = false;
  };

  const triggerScreenshot = () => {
    if (wsRef.current && isConnected) {
      const commandMessage = JSON.stringify({
        type: "command",
        command: "screenshot",
      });
      wsRef.current.send(commandMessage);
      console.log("[Frontend] Sent screenshot command");
    }
  };

  const onScroll = (e) => {
    const { layoutMeasurement, contentOffset, contentSize } = e.nativeEvent;
    const atBottom =
      contentOffset.y + layoutMeasurement.height >= contentSize.height - 12;
    userHoldingRef.current = !atBottom;
  };

  return (
    <View style={{ ...styles.container, paddingHorizontal: 24 }}>
      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          value={wsUrl}
          onChangeText={setWsUrl}
          placeholder={`ws://${address}:4000/stream?role=viewer`}
          autoCapitalize="none"
          autoCorrect={false}
          editable={!isConnected}
        />
        {!isConnected ? (
          <Button title="Connect" onPress={connect} />
        ) : (
          <>
            <TouchableOpacity
              style={styles.screenshotButton}
              onPress={triggerScreenshot}
              accessible={true}
              accessibilityLabel="Take screenshot"
            >
              <Icon name="photo-camera" size={24} color="#fff" />
            </TouchableOpacity>
            <TouchableOpacity
              style={styles.clearButton}
              onPress={clearContent}
              accessible={true}
              accessibilityLabel="Clear content"
            >
              <Icon name="delete" size={24} color="#fff" />
            </TouchableOpacity>
            <Button
              title="Disconnect"
              onPress={disconnect}
              color={Platform.OS === "ios" ? "red" : undefined}
            />
          </>
        )}
      </View>

      <ScrollView
        ref={scrollRef}
        style={styles.scroll}
        onScroll={onScroll}
        scrollEventThrottle={16}
        contentContainerStyle={styles.scrollInner}
      >
        <MDViewer markdown={content || (isConnected ? "" : "Not connected")} />
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, paddingTop: 48, backgroundColor: "#e7e7e7ff" },
  inputRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    paddingHorizontal: 12,
    marginBottom: 12,
  },
  input: {
    flex: 1,
    borderWidth: 1,
    borderColor: "#333",
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    color: "#000",
    backgroundColor: "#e7e7e7ff",
  },
  screenshotButton: {
    width: 44,
    height: 44,
    borderRadius: 8,
    backgroundColor: "#4CAF50",
    justifyContent: "center",
    alignItems: "center",
    borderWidth: 1,
    borderColor: "#4CAF50",
  },
  clearButton: {
    width: 44,
    height: 44,
    borderRadius: 8,
    backgroundColor: "#ef4d4dff",
    justifyContent: "center",
    alignItems: "center",
    borderWidth: 1,
    borderColor: "#ef4d4dff",
  },
  scroll: { flex: 1 },
  scrollInner: { paddingHorizontal: 12, paddingBottom: 24 },
  text: {
    color: "#e6e6e6",
    fontSize: 16,
    lineHeight: 22,
    fontFamily: Platform.select({ ios: "Menlo", android: "monospace" }),
  },
});
