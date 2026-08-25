"use strict";

const { spawn } = require("child_process");

class LSPClient {
  constructor(command, args, notification, log) {
    this.command = command;
    this.args = args;
    this.notification = notification;
    this.log = log;
    this.nextID = 1;
    this.pending = new Map();
    this.buffer = Buffer.alloc(0);
    this.process = null;
    this.stopping = false;
  }

  async start(initializeParams) {
    this.log(`Starting ${this.command} ${this.args.join(" ")}`);
    const child = spawn(this.command, this.args, { stdio: ["pipe", "pipe", "pipe"] });
    this.process = child;
    child.stdout.on("data", (chunk) => this.consume(chunk));
    child.stderr.on("data", (chunk) => this.log(chunk.toString().trimEnd()));
    child.on("exit", (code, signal) => this.closed(code, signal));
    await new Promise((resolve, reject) => {
      child.once("spawn", resolve);
      child.once("error", reject);
    });
    const result = await this.request("initialize", initializeParams);
    this.notify("initialized", {});
    return result;
  }

  request(method, params) {
    if (!this.process || !this.process.stdin.writable) {
      return Promise.reject(new Error("Northframe language server is not running"));
    }
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      try {
        this.send({ jsonrpc: "2.0", id, method, params });
      } catch (error) {
        this.pending.delete(id);
        reject(error);
      }
    });
  }

  notify(method, params) {
    if (!this.process || !this.process.stdin.writable) return;
    this.send({ jsonrpc: "2.0", method, params });
  }

  send(message) {
    const payload = JSON.stringify(message);
    const header = `Content-Length: ${Buffer.byteLength(payload, "utf8")}\r\n\r\n`;
    this.process.stdin.write(header + payload, "utf8");
  }

  consume(chunk) {
    this.buffer = Buffer.concat([this.buffer, chunk]);
    while (true) {
      const boundary = this.buffer.indexOf("\r\n\r\n");
      if (boundary < 0) return;
      const header = this.buffer.subarray(0, boundary).toString("ascii");
      const match = /^Content-Length:\s*(\d+)$/im.exec(header);
      if (!match) {
        this.log(`Invalid language server frame: ${header}`);
        this.buffer = this.buffer.subarray(boundary + 4);
        continue;
      }
      const length = Number(match[1]);
      const start = boundary + 4;
      if (this.buffer.length < start + length) return;
      const payload = this.buffer.subarray(start, start + length).toString("utf8");
      this.buffer = this.buffer.subarray(start + length);
      try {
        this.received(JSON.parse(payload));
      } catch (error) {
        this.log(`Invalid language server JSON: ${error.message}`);
      }
    }
  }

  received(message) {
    if (Object.prototype.hasOwnProperty.call(message, "id") && !message.method) {
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.error) pending.reject(new Error(message.error.message || "Language server request failed"));
      else pending.resolve(message.result);
      return;
    }
    if (message.method && !Object.prototype.hasOwnProperty.call(message, "id")) {
      this.notification(message.method, message.params);
      return;
    }
    if (message.method && Object.prototype.hasOwnProperty.call(message, "id")) {
      this.send({ jsonrpc: "2.0", id: message.id, error: { code: -32601, message: "Client method not supported" } });
    }
  }

  closed(code, signal) {
    const expected = this.stopping;
    this.process = null;
    const message = `Northframe language server exited${signal ? ` with ${signal}` : ` with code ${code}`}`;
    this.log(message);
    for (const pending of this.pending.values()) pending.reject(new Error(message));
    this.pending.clear();
    if (!expected) this.notification("northframe/serverExited", { message });
  }

  async stop() {
    if (!this.process) return;
    this.stopping = true;
    const child = this.process;
    try {
      await Promise.race([
        this.request("shutdown", null),
        new Promise((resolve) => setTimeout(resolve, 750)),
      ]);
      this.notify("exit", null);
    } catch (_) {
      // The process may already be gone.
    }
    setTimeout(() => {
      if (this.process === child) child.kill();
    }, 500).unref();
  }
}

module.exports = { LSPClient };
