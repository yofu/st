"use strict";

importScripts("workspace.js", "wasm_exec.js");

let started = false;
self.onmessage = async ({data: {model, files}}) => {
  if (started) return;
  started = true;
  const workspace = self.stWorkspace;
  try {
    workspace.apply(files);
    const original = new Map(workspace.entries().map(entry => [entry.path, entry.version]));
    self.stWorkerLog = text => postMessage({type: "log", text});
    self.stWorkerProgress = (phase, step, total) => postMessage({type: "progress", phase, step, total});
    self.stWorkerDone = (result, error) => {
      try {
        if (error) {
          postMessage({type: "done", error});
          return;
        }
        const entries = workspace.entries();
        const remaining = new Set(entries.map(entry => entry.path));
        const removed = [...original.keys()].filter(path => !remaining.has(path));
        const changed = entries.filter(entry => entry.version !== original.get(entry.path));
        const transfer = [result.buffer];
        for (const entry of changed) {
          if (!entry.directory) {
            entry.data = workspace.get(entry.path);
            transfer.push(entry.data.buffer);
          }
        }
        postMessage({type: "done", result, changed, removed}, transfer);
      } catch (e) {
        postMessage({type: "done", error: e.message});
      }
    };
    self.stWorkerReady = run => {
      postMessage({type: "progress", phase: "解析モデルを読み込み中", step: 0, total: 0});
      setTimeout(() => run(model), 0);
    };
    const go = new Go();
    go.env = {HOME: "/workspace", TMPDIR: "/tmp"};
    const response = await fetch("analysis.wasm");
    if (!response.ok) throw new Error(`analysis.wasm: HTTP ${response.status}`);
    const {instance} = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
    await go.run(instance);
    postMessage({type: "done", error: "解析Workerが終了しました"});
  } catch (error) {
    postMessage({type: "done", error: error.message});
  }
};
