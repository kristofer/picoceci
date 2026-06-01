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
    return state.replSessionId;
  }

  async function openFile(path) {
    const payload = await requestJSON(`/api/files/${encodePath(path)}`);
    setCurrentFile(payload.path);
    const editor = document.getElementById("editor-content");
    if (editor) {
      editor.value = payload.content;
      editor.focus();
    }
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

  document.addEventListener("DOMContentLoaded", function () {
    document.getElementById("save-button")?.addEventListener("click", function () {
      saveCurrentFile().catch((error) => setStatus(error.message));
    });

    document.getElementById("run-button")?.addEventListener("click", function () {
      runCurrentBuffer().catch((error) => appendConsole({ error: error.message }));
    });

    document.getElementById("repl-form")?.addEventListener("submit", function (event) {
      event.preventDefault();
      const input = document.getElementById("repl-input");
      const code = input.value.trim();
      if (!code) {
        return;
      }
      evalREPL(code)
        .then(function () {
          input.value = "";
          input.focus();
        })
        .catch((error) => appendConsole({ error: error.message }));
    });

    document.addEventListener("keydown", function (event) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        saveCurrentFile().catch((error) => setStatus(error.message));
      }
      if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
        event.preventDefault();
        runCurrentBuffer().catch((error) => appendConsole({ error: error.message }));
      }
    });
  });

  window.picoceciIDE = {
    openFile,
  };
})();
