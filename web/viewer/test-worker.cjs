const fs = require("node:fs");
const path = require("node:path");
const {Worker} = require("node:worker_threads");
const [site, testWasm, model] = process.argv.slice(2).map(value => path.resolve(value));
globalThis.crypto = require("node:crypto").webcrypto;
require(path.join(site, "workspace.js"));
require(path.join(site, "wasm_exec.js"));
stWorkspace.put(path.basename(model), fs.readFileSync(model));
const wrapper = `
const fs = require("node:fs");
const path = require("node:path");
const {parentPort, workerData} = require("node:worker_threads");
globalThis.self = globalThis;
globalThis.crypto = require("node:crypto").webcrypto;
globalThis.importScripts = (...files) => files.forEach(file => require(path.join(workerData, file)));
globalThis.fetch = async file => new Response(fs.readFileSync(path.join(workerData, file)));
globalThis.postMessage = (value, transfers) => parentPort.postMessage(value, transfers);
require(path.join(workerData, "analysis-worker.js"));
parentPort.on("message", data => globalThis.onmessage({data}));
`;
let current;
globalThis.stTestWorkerStart = (id, model, done, progress) => {
  const worker = new Worker(wrapper, {eval: true, workerData: site});
  current = {id, worker, done};
  const files = stWorkspace.entries();
  const transfer = [model.buffer];
  for (const entry of files) {
    if (!entry.directory) {
      entry.data = stWorkspace.get(entry.path);
      transfer.push(entry.data.buffer);
    }
  }
  worker.on("message", data => {
    if (current?.worker !== worker) return;
    if (data.type === "progress") progress(data.phase);
    if (data.type === "log") process.stdout.write(data.text);
    if (data.type === "done") {
      current = undefined;
      worker.terminate();
      if (!data.error) stWorkspace.apply(data.changed, data.removed);
      else console.error(data.error);
      done(id, data.result || null, data.error || "");
    }
  });
  worker.on("error", error => {
    if (current?.worker !== worker) return;
    current = undefined;
    console.error(error);
    done(id, null, error.message);
  });
  worker.postMessage({model, files}, transfer);
};
globalThis.stTestWorkerStop = id => {
  if (current?.id !== id) return;
  const job = current;
  current = undefined;
  job.worker.terminate();
  setImmediate(() => job.done(id, null, "stopped"));
};
const go = new Go();
go.env = {HOME:"/workspace", TMPDIR:"/tmp", ST_VIEWER_TEST_INP:"/workspace/" + path.basename(model)};
go.argv = ["worker.test", "-test.v", "-test.run=^TestViewerDedicatedWorker$", "-test.timeout=150s"];
go.exit = process.exit;
WebAssembly.instantiate(fs.readFileSync(testWasm), go.importObject).then(({instance}) => go.run(instance)).catch(err => { console.error(err); process.exit(1); });
