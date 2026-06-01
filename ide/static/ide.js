(function () {
  const state = {
    currentFile: "",
    replSessionId: "",
  };

  function encodePath(filePath) {
    return filePath
      .split("/")
      .filter(Boolean)
      .map(encodeURIComponent)
      .join("/");
  }

  async function requestJSON(url, options) {
    const response = await fetch(url, options);
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(payload.error || `request failed: ${response.status}`);
    }
    return payload;
  }

  function setStatus(message) {
    const status = document.getElementById("status-message");
    if (status) {
      status.textContent = message;
    }
  }

  function setCurrentFile(path) {
    state.currentFile = path;
    const label = document.getElementById("current-file-label");
    if (label) {
      label.textContent = path || "No file selected";
    }
  }

  function setSessionBadge() {
    const badge = document.getElementById("session-badge");
    if (!badge) {
      return;
    }
    badge.textContent = state.replSessionId ? `Session: ${state.replSessionId.slice(0, 8)}` : "Session: none";
  }

  function appendConsole(entry) {
    const consoleOutput = document.getElementById("console-output");
    if (!consoleOutput) {
      return;
    }
    if (entry.output) {
      const pre = document.createElement("pre");
      pre.className = "console__entry";
      pre.textContent = entry.output;
      consoleOutput.appendChild(pre);
    }
    if (entry.result) {
      const pre = document.createElement("pre");
      pre.className = "console__entry console__entry--result";
      pre.textContent = `=> ${entry.result}`;
      consoleOutput.appendChild(pre);
    }
    if (entry.error) {
      const pre = document.createElement("pre");
      pre.className = "console__entry console__entry--error";
      pre.textContent = entry.error;
      consoleOutput.appendChild(pre);
    }
    if (!entry.output && !entry.result && !entry.error) {
      const pre = document.createElement("pre");
      pre.className = "console__entry console__entry--muted";
      pre.textContent = "(no output)";
      consoleOutput.appendChild(pre);
    }
    consoleOutput.scrollTop = consoleOutput.scrollHeight;
  }

  async function ensureSession() {
    if (state.replSessionId) {
      return state.replSessionId;
    }
    const payload = await requestJSON("/api/repl/create", {
      method: "POST",
    });
    state.replSessionId = payload.id;
    setSessionBadge();
    return state.replSessionId;
  }

  async function resetSession() {
    if (state.replSessionId) {
      const response = await fetch(`/api/repl/${state.replSessionId}`, {
        method: "DELETE",
      });
      if (!response.ok && response.status !== 404) {
        throw new Error(`request failed: ${response.status}`);
      }
    }
    state.replSessionId = "";
    setSessionBadge();
    await ensureSession();
  }

  async function loadOutline(path) {
    const panel = document.getElementById("outline-panel");
    if (!panel) {
      return;
    }
    if (!path) {
      panel.textContent = "Open a file to load outline.";
      return;
    }
    try {
      const symbols = await requestJSON(`/api/project/outline/${encodePath(path)}`);
      if (!Array.isArray(symbols) || symbols.length === 0) {
        panel.textContent = "No symbols found.";
        return;
      }
      panel.innerHTML = "";
      const list = document.createElement("ul");
      list.className = "outline-list";
      symbols.forEach((symbol) => {
        const item = document.createElement("li");
        item.textContent = symbol;
        list.appendChild(item);
      });
      panel.appendChild(list);
    } catch {
      panel.textContent = "Outline unavailable for this file.";
    }
  }

  async function openFile(path) {
    const payload = await requestJSON(`/api/files/${encodePath(path)}`);
    setCurrentFile(payload.path);
    const editor = document.getElementById("editor-content");
    if (editor) {
      editor.value = payload.content;
      editor.focus();
    }
    await loadOutline(payload.path);
    setStatus(`Loaded ${payload.path}`);
  }

  async function saveCurrentFile() {
    if (!state.currentFile) {
      setStatus("Select a file before saving.");
      return;
    }
    const editor = document.getElementById("editor-content");
    await requestJSON(`/api/files/${encodePath(state.currentFile)}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ content: editor.value }),
    });
    setStatus(`Saved ${state.currentFile}`);
  }

  async function runCurrentBuffer() {
    const editor = document.getElementById("editor-content");
    const payload = await requestJSON("/api/execute", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ code: editor.value }),
    });
    appendConsole(payload);
    setStatus("Execution complete");
  }

  async function evalREPL(code) {
    const sessionID = await ensureSession();
    const payload = await requestJSON(`/api/repl/${sessionID}/eval`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ code }),
    });
    appendConsole(payload);
  }

  function copyCurrentSelection() {
    const editor = document.getElementById("editor-content");
    if (!(editor instanceof HTMLTextAreaElement)) {
      return Promise.resolve();
    }
    const selectedText = editor.value.slice(editor.selectionStart, editor.selectionEnd);
    const toCopy = selectedText || editor.value;
    return navigator.clipboard.writeText(toCopy).then(function () {
      setStatus(selectedText ? "Selection copied" : "Buffer copied");
    });
  }

  function evalInputBuffer() {
    const input = document.getElementById("repl-input");
    if (!(input instanceof HTMLInputElement)) {
      return Promise.resolve();
    }
    const code = input.value.trim();
    if (!code) {
      return Promise.resolve();
    }
    return evalREPL(code).then(function () {
      input.value = "";
      input.focus();
    });
  }

  function isMetaShortcut(event, key) {
    return (event.metaKey || event.ctrlKey) && event.key.toLowerCase() === key;
  }

  document.addEventListener("DOMContentLoaded", function () {
    setSessionBadge();

    document.getElementById("save-button")?.addEventListener("click", function () {
      saveCurrentFile().catch((error) => setStatus(error.message));
    });

    document.getElementById("copy-button")?.addEventListener("click", function () {
      copyCurrentSelection().catch((error) => setStatus(error.message));
    });

    document.getElementById("run-button")?.addEventListener("click", function () {
      runCurrentBuffer().catch((error) => appendConsole({ error: error.message }));
    });

    document.getElementById("new-session-button")?.addEventListener("click", function () {
      resetSession()
        .then(function () {
          setStatus("New REPL session ready");
        })
        .catch((error) => appendConsole({ error: error.message }));
    });

    document.getElementById("file-tree")?.addEventListener("click", function (event) {
      const target = event.target;
      if (!(target instanceof HTMLElement)) {
        return;
      }
      const filePath = target.dataset.path;
      if (!filePath) {
        return;
      }
      openFile(filePath).catch((error) => setStatus(error.message));
    });

    document.getElementById("repl-form")?.addEventListener("submit", function (event) {
      event.preventDefault();
      evalInputBuffer().catch((error) => appendConsole({ error: error.message }));
    });

    document.addEventListener("keydown", function (event) {
      if (isMetaShortcut(event, "s")) {
        event.preventDefault();
        saveCurrentFile().catch((error) => setStatus(error.message));
      }
      if (isMetaShortcut(event, "r")) {
        event.preventDefault();
        if (document.activeElement?.id === "repl-input") {
          evalInputBuffer().catch((error) => appendConsole({ error: error.message }));
          return;
        }
        runCurrentBuffer().catch((error) => appendConsole({ error: error.message }));
      }
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        event.preventDefault();
        runCurrentBuffer().catch((error) => appendConsole({ error: error.message }));
      }
      if (isMetaShortcut(event, "l")) {
        event.preventDefault();
        const input = document.getElementById("repl-input");
        if (input instanceof HTMLInputElement) {
          input.focus();
        }
      }
    });
  });

  window.picoceciIDE = {
    openFile,
  };
})();
