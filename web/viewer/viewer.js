"use strict";
(() => {
  const $ = id => document.getElementById(id);
  const file = $("file"), open = $("open"), save = $("save"), download = $("download");
  const status = $("status"), selection = $("selection"), hint = selection.textContent;
  const field = $("command"), log = $("history"), latest = $("latest"), prompt = $("prompt");
  const actions = Array.from(document.querySelectorAll("[data-action]"));
  const limit = 32 * 1024 * 1024;
  let importModel, dispatch, submit, exportModel, reply, answer, modelText = "", pendingName = "";
  const workspace = globalThis.stWorkspace;
  const questionDialog = $("question-dialog");
  let question;
  const workspaceDialog = $("workspace-dialog"), textDialog = $("text-dialog");
  let workspacePrefix = "", textPath = "", annotations = "", workspaceURL;
  let analysisJob, finishAnalysis, analysisLog;
  let busy = false, hasModel = false, editing = false, modified = false, promptID = 0, downloadURL;
  const historyKey = "st-viewer-commands-v1";
  let entries = [], historyIndex = 0, draft = "";
  try {
    const saved = JSON.parse(localStorage.getItem(historyKey) || "[]");
    if (Array.isArray(saved)) entries = saved.filter(s => typeof s === "string" && s.length <= 1024).slice(-100);
  } catch { /* Storage may be disabled. */ }
  historyIndex = entries.length;
  function message(text, error = false) { status.textContent = text; status.dataset.error = String(error); }
  function controls() {
    open.disabled = busy || !!analysisJob || !importModel;
    save.disabled = busy || !!analysisJob || !hasModel || editing;
    $("stage").disabled = !!analysisJob;
    $("stop-analysis").hidden = !analysisJob;
    $("stop-analysis").disabled = Boolean(analysisJob?.finishing);
    $("analysis-meter").hidden = !analysisJob;
    $("workspace").disabled = !workspace;
    for (const button of actions) button.disabled = busy || !hasModel;
    for (const id of ["command", "execute", "commands", "previous", "next"]) $(id).disabled = busy || !submit;
    $("edit-tools").hidden = !editing;
  }
  function setBusy(value) { busy = value; controls(); }
  function clearDownload() {
    if (downloadURL) URL.revokeObjectURL(downloadURL);
    downloadURL = undefined;
    download.removeAttribute("href");
    download.hidden = true;
  }
  window.stViewerReady = (load, action, command, saveFile, replyToPrompt, names, answerQuestion, finishWorker, logWorker) => {
    importModel = load; dispatch = action; submit = command; exportModel = saveFile; reply = replyToPrompt; answer = answerQuestion; finishAnalysis = finishWorker; analysisLog = logWorker;
    for (const name of names) $("commands").add(new Option(name, name));
    controls(); message("INPファイルを選択してください。");
  };
  window.stViewerState = (name, nodes, members, selectedNodes, selectedMembers, active, changed, history, id, title, initial, progress, nextAnnotations, nextQuestion) => {
    hasModel = Boolean(name);
    editing = active;
    modified = changed;
    if (!analysisJob) $("progress").textContent = progress || "";
    if (annotations !== nextAnnotations) {
      annotations = nextAnnotations;
      const overlay = $("annotations");
      overlay.replaceChildren();
      for (const item of JSON.parse(annotations || "[]")) {
        const div = document.createElement("div");
        div.style.left = `${item.X}px`; div.style.top = `${item.Y}px`;
        div.textContent = item.Text;
        overlay.append(div);
      }
    }
    if (!nextQuestion) {
      question = undefined;
      if (questionDialog.open) questionDialog.close();
    } else {
      const incoming = JSON.parse(nextQuestion);
      if (incoming.ID !== question?.ID) {
        question = incoming;
        $("question-title").textContent = question.Title;
        $("question-message").textContent = question.Message;
        const choices = $("question-choices");
        choices.replaceChildren();
        question.Choices.forEach((label, index) => {
          const button = document.createElement("button");
          button.textContent = label;
          button.addEventListener("click", () => resolveQuestion(index + 1));
          choices.append(button);
        });
        if (!questionDialog.open) questionDialog.showModal();
      }
    }
    modelText = name ? `${name}${changed ? " *" : ""} — 節点 ${nodes}・部材 ${members}` : "";
    if (!busy && name && status.dataset.error !== "true" && download.hidden) message(modelText);
    selection.textContent = `${hint} ｜ 選択: 節点 ${selectedNodes}・部材 ${selectedMembers}`;
    if (log.textContent !== history) {
      log.textContent = history;
      log.scrollTop = log.scrollHeight;
      const text = history.trimEnd();
      latest.textContent = text.slice(text.lastIndexOf("\n") + 1);
    }
    if (id !== promptID) {
      promptID = id;
      if (id) {
        $("prompt-title").textContent = title;
        $("prompt-value").value = initial;
        if (!prompt.open) prompt.showModal();
        $("prompt-value").select();
      } else if (prompt.open) prompt.close();
    }
    controls();
  };
  function resolveQuestion(choice) {
    if (question && answer?.(question.ID, choice)) questionDialog.close();
  }
  questionDialog.addEventListener("cancel", event => {
    event.preventDefault();
    if (question) resolveQuestion(question.Choices.length);
  });
  window.stViewerLoaded = error => {
    setBusy(false);
    message(error || `${pendingName} を読み込みました`, Boolean(error));
  };
  window.stViewerExported = (name, bytes, error) => {
    setBusy(false);
    if (error) { message(error, true); return; }
    clearDownload();
    try { workspace.put(name, bytes); }
    catch (error) { console.warn(error.message); }
    downloadURL = URL.createObjectURL(new Blob([bytes], {type: "application/octet-stream"}));
    download.href = downloadURL; download.download = name; download.hidden = false;
    message("書き出しました。「INPをダウンロード」をタップしてください。");
  };
  window.stViewerFatal = error => { message(`起動できません: ${error}`, true); setBusy(true); };
  open.addEventListener("click", () => file.click());
  file.addEventListener("change", async () => {
    const selected = file.files[0]; file.value = "";
    if (!selected || busy || analysisJob || !importModel) return;
    if (!/\.inp$/i.test(selected.name) || !selected.size || selected.size > limit) {
      message("32 MiB以内の空でない .inp ファイルを選択してください。", true); return;
    }
    if (modified && !window.confirm("現在の編集内容を置き換えて、別のINPを開きますか？")) return;
    clearDownload(); setBusy(true); pendingName = selected.name;
    message(`${selected.name} を読み込み中…`);
    try {
      const bytes = new Uint8Array(await selected.arrayBuffer());
      if (!importModel(selected.name, bytes)) throw new Error("読込みを受け付けられませんでした");
    } catch (error) { setBusy(false); message(error.message, true); }
  });
  function saveINP(filename = "") {
    if (!exportModel || busy || analysisJob) return false;
    if (!hasModel || editing) {
      message(hasModel ? "対話操作を確定または中止してから保存してください。" : "先にINPファイルを開いてください。", true);
      return false;
    }
    clearDownload(); setBusy(true);
    if (!exportModel(filename)) {
      setBusy(false); message("保存を受け付けられませんでした。", true);
      return false;
    }
    return true;
  }
  save.addEventListener("click", () => saveINP());
  function refreshWorkspace() {
    const list = $("workspace-files");
    list.replaceChildren();
    const files = workspace.list().filter(file => file.path.startsWith(workspacePrefix));
    for (const file of files) {
      const item = document.createElement("li");
      const link = document.createElement("a");
      link.textContent = `${file.path} (${file.size.toLocaleString()} B)`;
      link.download = file.path.split("/").at(-1);
      link.href = "#";
      link.addEventListener("click", event => {
        try {
          if (workspaceURL) URL.revokeObjectURL(workspaceURL);
          workspaceURL = URL.createObjectURL(new Blob([workspace.get(file.path)]));
          link.href = workspaceURL;
        } catch (error) {
          event.preventDefault(); $("workspace-status").textContent = error.message;
        }
      });
      item.append(link);
      list.append(item);
    }
    if (!files.length) $("workspace-status").textContent = "作業フォルダにファイルがありません。";
  }
  function openWorkspace(prefix = "") {
    workspacePrefix = prefix.replace(/^\/workspace(?:\/|$)/, "").replace(/^\.\/?$/, "").replace(/\/$/, "");
    if (workspacePrefix) workspacePrefix += "/";
    $("workspace-status").textContent = "";
    refreshWorkspace();
    if (!workspaceDialog.open) workspaceDialog.showModal();
  }
  $("workspace").addEventListener("click", () => openWorkspace());
  $("refresh-files").addEventListener("click", () => { $("workspace-status").textContent = ""; refreshWorkspace(); });
  $("close-workspace").addEventListener("click", () => workspaceDialog.close());
  $("stage").addEventListener("click", () => $("related-files").click());
  $("related-files").addEventListener("change", async event => {
    const selected = Array.from(event.target.files);
    event.target.value = "";
    if (analysisJob) return;
    const errors = [];
    for (const file of selected) {
      try {
        if (file.size > limit) throw new Error("32 MiBを超えています");
        if (workspace.list().some(existing => existing.path === file.name) &&
            !window.confirm(`${file.name} を作業フォルダ内で上書きしますか？`)) continue;
        workspace.put(file.name, new Uint8Array(await file.arrayBuffer()));
      } catch (error) { errors.push(`${file.name} ${error.message}`); }
    }
    workspacePrefix = "";
    $("workspace-status").textContent = errors.join(" / ");
    refreshWorkspace();
  });
  function openText(path, clipboardText) {
    textPath = path;
    const editor = $("artifact-text");
    const readOnly = clipboardText !== undefined;
    editor.readOnly = readOnly;
    $("text-form").querySelector('button[type="submit"]').hidden = readOnly;
    $("text-title").textContent = readOnly ? "出力テキスト（選択してコピー）" : path;
    $("text-status").textContent = "";
    if (readOnly) editor.value = clipboardText;
    else {
      try { editor.value = new TextDecoder("utf-8", {fatal: true}).decode(workspace.get(path)); }
      catch (error) {
        if (error.code !== "ENOENT") { message(error.message, true); return; }
        editor.value = "";
      }
    }
    if (!textDialog.open) textDialog.showModal();
    if (readOnly) editor.select();
  }
  $("close-text").addEventListener("click", () => textDialog.close());
  $("text-form").addEventListener("submit", event => {
    event.preventDefault();
    if ($("artifact-text").readOnly || analysisJob) return;
    try {
      workspace.writeText(textPath, $("artifact-text").value);
      $("text-status").textContent = "作業フォルダに保存しました。";
    } catch (error) { $("text-status").textContent = error.message; }
  });
  window.stViewerBrowserAction = (action, paths) => {
    // Wait for the current WebAssembly callback to return before opening a dialog.
    queueMicrotask(() => {
      switch (action) {
      case "workspace": openWorkspace(paths[0]); break;
      case "text": openText(paths[0]); break;
      case "clipboard": openText("", paths[0]); break;
      case "print": requestAnimationFrame(() => requestAnimationFrame(() => window.print())); break;
      case "download": openWorkspace(); break;
      }
    });
  };
  function updateAnalysisProgress(job) {
    if (analysisJob !== job) return;
    const elapsed = Math.floor((performance.now() - job.started) / 1000);
    const units = job.total > 0 ? ` ${job.step} / ${job.total}` : job.step > 0 ? ` ${job.step}` : "";
    $("progress").textContent = `${job.phase}${units}（${elapsed}秒）`;
    const meter = $("analysis-meter");
    if (job.total > 0 && !job.finishing) {
      meter.max = job.total;
      meter.value = Math.min(job.step, job.total);
    } else meter.removeAttribute("value");
  }
  function finishWorker(job, data = {}) {
    if (job !== analysisJob || job.finishing) return;
    job.finishing = true;
    job.worker?.terminate();
    job.phase = data.error || "解析結果を反映中";
    job.files = data.changed || [];
    job.removed = data.removed || [];
    updateAnalysisProgress(job);
    controls();
    function deliver() {
      if (job !== analysisJob) return;
      if (!finishAnalysis(job.id, data.result || null, data.error || "")) setTimeout(deliver, 50);
    }
    deliver();
  }
  async function startWorker(job, model) {
    try {
      job.worker = new Worker("analysis-worker.js");
      job.worker.onmessage = ({data}) => {
        if (job !== analysisJob || job.finishing) return;
        switch (data.type) {
        case "progress":
          job.phase = {Completed: "結果をまとめています", Failed: "解析失敗",
            "Calculating...": "解析中", "Time history...": "時刻歴解析中"}[data.phase] || data.phase;
          job.step = data.step;
          job.total = data.total;
          updateAnalysisProgress(job);
          break;
        case "log": analysisLog(data.text); break;
        case "done": finishWorker(job, data); break;
        }
      };
      job.worker.onerror = event => {
        event.preventDefault();
        finishWorker(job, {error: event.message || "解析Workerを起動できません"});
      };
      job.worker.onmessageerror = () => finishWorker(job, {error: "解析結果を受信できません"});
      const files = workspace.entries();
      const transfer = [model.buffer];
      let copiedAt = performance.now();
      for (const entry of files) {
        if (job.finishing) return;
        if (!entry.directory) {
          entry.data = workspace.get(entry.path);
          transfer.push(entry.data.buffer);
        }
        if (performance.now() - copiedAt > 12) {
          await new Promise(resolve => setTimeout(resolve, 0));
          copiedAt = performance.now();
        }
      }
      if (!job.finishing) job.worker.postMessage({model, files}, transfer);
    } catch (error) {
      finishWorker(job, {error: error.message});
    }
  }
  window.stViewerWorkerStart = (id, model) => {
    const job = {id, started: performance.now(), phase: "Workerを準備中", step: 0, total: 0};
    analysisJob = job;
    job.timer = setInterval(() => updateAnalysisProgress(job), 1000);
    controls();
    updateAnalysisProgress(job);
    setTimeout(() => { if (!job.finishing) startWorker(job, model); }, 0);
  };
  window.stViewerWorkerStop = id => {
    const job = analysisJob;
    if (job && job.id === id) queueMicrotask(() => finishWorker(job, {error: "解析を停止しました"}));
  };
  window.stViewerWorkerApplied = (id, error) => {
    const job = analysisJob;
    if (!job || job.id !== id) return;
    clearInterval(job.timer);
    if (!error) {
      try { workspace.apply(job.files, job.removed); }
      catch (e) { error = e.message; }
    }
    analysisJob = undefined;
    if (workspaceDialog.open) refreshWorkspace();
    if (error) message(error, true);
    controls();
  };
  $("stop-analysis").addEventListener("click", () => {
    if (analysisJob) window.stViewerWorkerStop(analysisJob.id);
  });

  function browse(delta) {
    if (historyIndex === entries.length) draft = field.value;
    historyIndex = Math.max(0, Math.min(entries.length, historyIndex + delta));
    field.value = historyIndex === entries.length ? draft : entries[historyIndex];
  }
  $("previous").addEventListener("click", () => browse(-1));
  $("next").addEventListener("click", () => browse(1));
  field.addEventListener("input", event => {
    if (!event.isComposing && field.value.startsWith(";")) field.setRangeText(":", 0, 1, "preserve");
  });
  field.addEventListener("keydown", event => {
    if (event.isComposing) return;
    if (event.key === "ArrowUp" || event.key === "ArrowDown") {
      event.preventDefault(); browse(event.key === "ArrowUp" ? -1 : 1);
    } else if (event.key === "Escape") { event.preventDefault(); dispatch?.("cancel"); field.blur(); }
  });
  $("commands").addEventListener("change", event => {
    if (event.target.value) field.value = event.target.value;
    event.target.value = "";
  });
  $("command-form").addEventListener("submit", event => {
    event.preventDefault();
    const text = field.value.trim().replace(/^;/, ":");
    if (!text || !submit || busy) return;
    const openCommand = /^:(?:e|ed|edi|edit)!?(?:\s+(.*))?$/i.exec(text);
    const saveCommand = /^:(?:w|wr|wri|writ|write|sav|save)!?(?:\s+(.*))?$/i.exec(text);
    if (openCommand && !openCommand[1] && !text.includes("|")) {
      if (analysisJob) { message("解析を完了または停止してからモデルを開いてください。", true); return; }
      // Safari requires file.click() to be part of the user gesture.
      file.click();
    } else if (saveCommand && !text.includes("|") &&
               !/(?:^|\s)-[a-z]|[\/\\]/i.test(saveCommand[1] || "")) {
      if (!saveINP(saveCommand[1])) return;
    } else {
      if (!submit(text)) { message("コマンドを受け付けられませんでした。", true); return; }
      clearDownload();
      if (modelText) message(modelText);
    }
    if (entries.at(-1) !== text) entries = [...entries, text].slice(-100);
    try { localStorage.setItem(historyKey, JSON.stringify(entries)); } catch { /* Optional. */ }
    historyIndex = entries.length; draft = ""; field.value = "";
    field.blur();
  });
  for (const button of actions) button.addEventListener("click", () => {
    const action = button.dataset.action;
    if (busy || !dispatch || !dispatch(action)) return;
    if (action === "member" || action === "node") {
      for (const other of actions.filter(b => b.hasAttribute("aria-pressed"))) other.setAttribute("aria-pressed", String(other === button));
    }
    if (modelText && download.hidden) message(modelText);
  });
  $("keys").addEventListener("change", event => {
    if (event.target.value) dispatch?.(`key:${event.target.value}`);
    event.target.value = "";
  });
  function replyPrompt(accepted) {
    if (promptID && reply?.(promptID, $("prompt-value").value, accepted)) prompt.close();
  }
  $("prompt-form").addEventListener("submit", event => { event.preventDefault(); replyPrompt(true); });
  $("prompt-cancel").addEventListener("click", () => replyPrompt(false));
  prompt.addEventListener("cancel", event => { event.preventDefault(); replyPrompt(false); });
  if (typeof ResizeObserver === "function") {
    new ResizeObserver(() => window.dispatchEvent(new Event("resize"))).observe($("giowindow"));
  }
  if (window.visualViewport) {
    const resize = () => document.documentElement.style.setProperty("--viewer-height", `${visualViewport.height}px`);
    visualViewport.addEventListener("resize", resize); resize();
  }
  async function start() {
    const go = new Go();
    go.env = {...go.env, HOME: "/workspace", TMPDIR: "/tmp"};
    const response = await fetch("stio.wasm");
    if (!response.ok) throw new Error(`stio.wasm: HTTP ${response.status}`);
    let result;
    if (WebAssembly.instantiateStreaming) {
      try { result = await WebAssembly.instantiateStreaming(response.clone(), go.importObject); }
      catch { /* Some local servers have no application/wasm MIME type. */ }
    }
    if (!result) result = await WebAssembly.instantiate(await response.arrayBuffer(), go.importObject);
    await go.run(result.instance);
  }
  start().catch(error => window.stViewerFatal(error.message));
})();
