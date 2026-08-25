const probePath = "/_northframe/dev";
const socketPath = "/_northframe/dev/reload";

export async function enableDevelopmentReload() {
  let response;
  try {
    response = await fetch(probePath, { method: "HEAD", cache: "no-store" });
  } catch {
    return;
  }
  if (!response.ok) return;

  let connected = false;
  let disconnected = false;
  let attempts = 0;

  const connect = () => {
    const scheme = window.location.protocol === "https:" ? "wss:" : "ws:";
    const socket = new WebSocket(`${scheme}//${window.location.host}${socketPath}`);

    socket.addEventListener("open", () => {
      attempts = 0;
      if (disconnected) {
        window.location.reload();
        return;
      }
      connected = true;
    });

    socket.addEventListener("close", () => {
      if (!connected) return;
      disconnected = true;
      attempts += 1;
      window.setTimeout(connect, Math.min(1000, 100 + attempts * 100));
    });
  };

  connect();
}
