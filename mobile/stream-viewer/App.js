import React, { useState, useEffect, useRef } from 'react';
import { View, TextInput, Button, StyleSheet, ScrollView, Text, Platform } from 'react-native';

export default function App() {
  const [wsUrl, setWsUrl] = useState('ws://10.0.0.33:4000/stream?role=viewer'); // <— change to your Mac's IP
  const [isConnected, setIsConnected] = useState(false);
  const [content, setContent] = useState('');

  const scrollRef = useRef(null);
  const userHoldingRef = useRef(false);
  const wsRef = useRef(null);

  // Simple coalescing buffer to reduce re-renders
  const pendingRef = useRef('');
  const rafRef = useRef(null);

  useEffect(() => {
    // Clean up WebSocket connection on unmount
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

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
      let chunk = '';
      try {
        // Parse the JSON message
        const data = typeof event.data === 'string' ? event.data : String(event.data);
        const parsed = JSON.parse(data);
        chunk = typeof parsed?.chunk === 'string' ? parsed.chunk : data; // fall back to raw
      } catch {
        // not JSON? append raw
        chunk = typeof event.data === 'string' ? event.data : String(event.data);
      }

      pendingRef.current += chunk;

      if (rafRef.current == null) {
        rafRef.current = requestAnimationFrame(() => {
          setContent((prev) => prev + pendingRef.current);
          pendingRef.current = '';
          rafRef.current = null;

          if (!userHoldingRef.current) {
            requestAnimationFrame(() => scrollRef.current?.scrollToEnd({ animated: true }));
          }
        });
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    ws.onerror = (error) => {
      console.log('WebSocket error:', error);
    };
  };

  const disconnect = () => {
    if (wsRef.current) {
      wsRef.current.close();
    }
    setIsConnected(false);
  };

  const onScroll = (e) => {
    const { layoutMeasurement, contentOffset, contentSize } = e.nativeEvent;
    const atBottom = contentOffset.y + layoutMeasurement.height >= contentSize.height - 12;
    userHoldingRef.current = !atBottom;
  };

  return (
    <View style={styles.container}>
      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          value={wsUrl}
          onChangeText={setWsUrl}
          placeholder="ws://10.0.0.33:4000/stream?role=viewer"
          autoCapitalize="none"
          autoCorrect={false}
          editable={!isConnected}
        />
        {!isConnected ? (
          <Button title="Connect" onPress={connect} />
        ) : (
          <Button title="Disconnect" onPress={disconnect} color={Platform.OS === 'ios' ? 'red' : undefined} />
        )}
      </View>

      <ScrollView
        ref={scrollRef}
        style={styles.scroll}
        onScroll={onScroll}
        scrollEventThrottle={16}
        contentContainerStyle={styles.scrollInner}
      >
        <Text style={styles.text}>{content || (isConnected ? '' : 'Not connected')}</Text>
      </ScrollView>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, paddingTop: 48, backgroundColor: '#0b0b0b' },
  inputRow: { flexDirection: 'row', gap: 8, paddingHorizontal: 12, marginBottom: 12 },
  input: {
    flex: 1,
    borderWidth: 1, borderColor: '#333', borderRadius: 8, paddingHorizontal: 12, paddingVertical: 10,
    color: '#fff', backgroundColor: '#141414',
  },
  scroll: { flex: 1 },
  scrollInner: { paddingHorizontal: 12, paddingBottom: 24 },
  text: { color: '#e6e6e6', fontSize: 16, lineHeight: 22, fontFamily: Platform.select({ ios: 'Menlo', android: 'monospace' }) },
});

// import { View, Text, StyleSheet } from 'react-native';

// export default function App() {
//   return (
//     <View style={styles.container}>
//       <Text style={styles.text}>Hello from Expo SDK 54 👋</Text>
//     </View>
//   );
// }

// const styles = StyleSheet.create({
//   container: { flex: 1, backgroundColor: '#0b0b0b', alignItems: 'center', justifyContent: 'center' },
//   text: { color: '#fff', fontSize: 20 },
// });
