// AgentSpec Frontend — Vanilla JS + SSE
(function () {
  "use strict";

  // --- State ---
  var state = {
    agents: [],
    currentAgent: "",
    apiKey: sessionStorage.getItem("agentspec_api_key") || "",
    streaming: false,
    sessionId: sessionStorage.getItem("agentspec_session_id") || "",
    eventSource: null,
  };

  // --- DOM refs ---
  var els = {};
  function initRefs() {
    els.messages = document.getElementById("messages");
    els.input = document.getElementById("input");
    els.btnSend = document.getElementById("btn-send");
    els.agentSelect = document.getElementById("agent-select");
    els.statusDot = document.getElementById("status-dot");
    els.activityLog = document.getElementById("activity-log");
    els.btnKey = document.getElementById("btn-key");
    els.keyModal = document.getElementById("key-modal");
    els.keyInput = document.getElementById("key-input");
    els.btnKeySave = document.getElementById("btn-key-save");
    els.btnKeyClear = document.getElementById("btn-key-clear");
    els.btnToggleActivity = document.getElementById("btn-toggle-activity");
    els.dynamicInputs = document.getElementById("dynamic-inputs");
  }

  // --- Helpers ---
  function authHeaders() {
    var h = { "Content-Type": "application/json" };
    if (state.apiKey) {
      h["Authorization"] = "Bearer " + state.apiKey;
    }
    return h;
  }

  function escapeHtml(text) {
    var div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }

  // --- Lightweight Markdown Parser ---
  function renderMarkdown(src) {
    var html = escapeHtml(src);

    // Fenced code blocks: ```lang\n...\n```
    html = html.replace(/```(\w*)\n([\s\S]*?)```/g, function (_, lang, code) {
      return '<pre><code class="lang-' + lang + '">' + code.replace(/^\n|\n$/g, '') + '</code></pre>';
    });

    // Inline code: `code`
    html = html.replace(/`([^`\n]+)`/g, '<code>$1</code>');

    // Headings: # to ####
    html = html.replace(/^#### (.+)$/gm, '<h4>$1</h4>');
    html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
    html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
    html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');

    // Horizontal rule
    html = html.replace(/^---+$/gm, '<hr>');

    // Blockquote
    html = html.replace(/^&gt; (.+)$/gm, '<blockquote>$1</blockquote>');

    // Bold + italic
    html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
    // Bold
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    // Italic
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

    // Strikethrough
    html = html.replace(/~~(.+?)~~/g, '<del>$1</del>');

    // Links: [text](url)
    html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');

    // Tables
    html = html.replace(/((?:^\|.+\|$\n?)+)/gm, function (tableBlock) {
      var rows = tableBlock.trim().split('\n');
      if (rows.length < 2) return tableBlock;
      var out = '<table>';
      rows.forEach(function (row, idx) {
        // Skip separator row (|---|---|)
        if (/^\|[\s\-:|]+\|$/.test(row)) return;
        var tag = idx === 0 ? 'th' : 'td';
        var cells = row.split('|').filter(function (c, i, a) { return i > 0 && i < a.length - 1; });
        out += '<tr>';
        cells.forEach(function (cell) {
          out += '<' + tag + '>' + cell.trim() + '</' + tag + '>';
        });
        out += '</tr>';
      });
      out += '</table>';
      return out;
    });

    // Unordered lists
    html = html.replace(/((?:^[\-\*] .+$\n?)+)/gm, function (block) {
      var items = block.trim().split('\n');
      var out = '<ul>';
      items.forEach(function (item) {
        out += '<li>' + item.replace(/^[\-\*] /, '') + '</li>';
      });
      out += '</ul>';
      return out;
    });

    // Ordered lists
    html = html.replace(/((?:^\d+\. .+$\n?)+)/gm, function (block) {
      var items = block.trim().split('\n');
      var out = '<ol>';
      items.forEach(function (item) {
        out += '<li>' + item.replace(/^\d+\. /, '') + '</li>';
      });
      out += '</ol>';
      return out;
    });

    // Paragraphs: wrap remaining loose text lines
    html = html.replace(/^(?!<[a-z])((?!<\/?\w).+)$/gm, '<p>$1</p>');

    // Clean up empty paragraphs and extra newlines
    html = html.replace(/<p>\s*<\/p>/g, '');
    html = html.replace(/\n{2,}/g, '\n');

    return html.trim();
  }

  function scrollToBottom() {
    els.messages.scrollTop = els.messages.scrollHeight;
  }

  // --- API ---
  function fetchAgents() {
    fetch("/v1/agents", { headers: authHeaders() })
      .then(function (r) {
        if (!r.ok) throw new Error("Failed to load agents");
        return r.json();
      })
      .then(function (data) {
        state.agents = data.agents || [];
        renderAgentSelect();
        setConnected(true);
      })
      .catch(function (err) {
        setConnected(false);
        addSystemMessage("Failed to connect: " + err.message);
      });
  }

  function createSession(agentName) {
    fetch("/v1/agents/" + encodeURIComponent(agentName) + "/sessions", {
      method: "POST",
      headers: authHeaders(),
      body: JSON.stringify({}),
    })
      .then(function (r) {
        if (!r.ok) throw new Error("Session create failed");
        return r.json();
      })
      .then(function (data) {
        state.sessionId = data.session_id;
        sessionStorage.setItem("agentspec_session_id", data.session_id);
      })
      .catch(function () {
        state.sessionId = "";
      });
  }

  // --- Rendering ---
  function renderAgentSelect() {
    els.agentSelect.innerHTML = "";
    if (state.agents.length === 0) {
      var opt = document.createElement("option");
      opt.value = "";
      opt.textContent = "No agents available";
      els.agentSelect.appendChild(opt);
      return;
    }
    state.agents.forEach(function (a) {
      var opt = document.createElement("option");
      opt.value = a.name;
      opt.textContent = a.name;
      if (a.name === state.currentAgent) opt.selected = true;
      els.agentSelect.appendChild(opt);
    });
    if (!state.currentAgent && state.agents.length > 0) {
      state.currentAgent = state.agents[0].name;
      els.agentSelect.value = state.currentAgent;
      createSession(state.currentAgent);
      loadAgentSchema(state.currentAgent);
    }
  }

  function setConnected(connected) {
    if (connected) {
      els.statusDot.classList.add("connected");
      els.statusDot.title = "Connected";
    } else {
      els.statusDot.classList.remove("connected");
      els.statusDot.title = "Disconnected";
    }
  }

  function addMessage(role, content) {
    var div = document.createElement("div");
    div.className = "msg " + role;
    div.textContent = content;
    els.messages.appendChild(div);
    scrollToBottom();
    return div;
  }

  function addSystemMessage(text) {
    addMessage("system", text);
  }

  function addActivityItem(type, content) {
    var div = document.createElement("div");
    div.className = "activity-item " + type;
    var label = document.createElement("div");
    label.className = "activity-label";
    label.textContent = type.replace("_", " ");
    var body = document.createElement("div");
    body.className = "activity-content";
    body.textContent = content;
    div.appendChild(label);
    div.appendChild(body);
    els.activityLog.appendChild(div);
    els.activityLog.scrollTop = els.activityLog.scrollHeight;
  }

  // --- Dynamic Input Controls (T070) ---
  function loadAgentSchema(agentName) {
    els.dynamicInputs.innerHTML = "";
    // Attempt to fetch agent details with input schema
    fetch("/v1/agents", { headers: authHeaders() })
      .then(function (r) { return r.json(); })
      .then(function (data) {
        var agents = data.agents || [];
        var agent = agents.find(function (a) { return a.name === agentName; });
        if (agent && agent.input_schema) {
          renderDynamicInputs(agent.input_schema);
        }
      })
      .catch(function () {});
  }

  function renderDynamicInputs(schema) {
    els.dynamicInputs.innerHTML = "";
    if (!schema || !schema.fields) return;

    schema.fields.forEach(function (field) {
      var wrapper = document.createElement("div");
      wrapper.style.cssText = "margin-bottom:8px;";

      var lbl = document.createElement("label");
      lbl.textContent = field.label || field.name;
      lbl.style.cssText = "display:block;font-size:12px;color:var(--muted);margin-bottom:4px;";
      wrapper.appendChild(lbl);

      var el;
      if (field.type === "select" && field.options) {
        el = document.createElement("select");
        field.options.forEach(function (opt) {
          var o = document.createElement("option");
          o.value = opt;
          o.textContent = opt;
          el.appendChild(o);
        });
      } else if (field.type === "file") {
        el = document.createElement("input");
        el.type = "file";
      } else {
        el = document.createElement("input");
        el.type = "text";
        el.placeholder = field.placeholder || "";
      }
      el.dataset.fieldName = field.name;
      el.className = "dynamic-field";
      el.style.cssText = "width:100%;background:var(--bg);color:var(--text);border:1px solid var(--border);border-radius:6px;padding:8px 10px;font-size:13px;";
      wrapper.appendChild(el);
      els.dynamicInputs.appendChild(wrapper);
    });
  }

  function collectDynamicInputValues() {
    var values = {};
    var fields = els.dynamicInputs.querySelectorAll(".dynamic-field");
    fields.forEach(function (f) {
      if (f.value) {
        values[f.dataset.fieldName] = f.value;
      }
    });
    return values;
  }

  // --- Send Message ---
  function sendMessage() {
    var text = els.input.value.trim();
    if (!text || state.streaming || !state.currentAgent) return;

    addMessage("user", text);
    els.input.value = "";
    els.input.style.height = "auto";
    state.streaming = true;
    els.btnSend.disabled = true;

    // Collect any dynamic input values as variables
    var variables = collectDynamicInputValues();

    // Create a placeholder for the assistant response
    var assistantDiv = addMessage("assistant", "");
    // Track raw markdown text for final rendering
    var streamCtx = { rawText: "" };

    // Open SSE connection for streaming
    var body = JSON.stringify({
      message: text,
      session_id: state.sessionId,
      variables: Object.keys(variables).length > 0 ? variables : undefined,
    });

    fetch("/v1/agents/" + encodeURIComponent(state.currentAgent) + "/stream", {
      method: "POST",
      headers: authHeaders(),
      body: body,
    })
      .then(function (response) {
        if (!response.ok) {
          return response.json().then(function (err) {
            throw new Error(err.message || "Request failed");
          });
        }

        var reader = response.body.getReader();
        var decoder = new TextDecoder();
        var buffer = "";

        function processChunk() {
          return reader.read().then(function (result) {
            if (result.done) {
              finishStreaming(assistantDiv, streamCtx);
              return;
            }

            buffer += decoder.decode(result.value, { stream: true });
            var lines = buffer.split("\n");
            buffer = lines.pop(); // Keep incomplete line in buffer

            var eventType = "";
            var eventData = "";

            for (var i = 0; i < lines.length; i++) {
              var line = lines[i];
              if (line.startsWith("event: ")) {
                eventType = line.substring(7);
              } else if (line.startsWith("data: ")) {
                eventData = line.substring(6);
                if (eventType && eventData) {
                  handleSSEEvent(eventType, eventData, assistantDiv, streamCtx);
                  eventType = "";
                  eventData = "";
                }
              } else if (line === "") {
                // Empty line = end of event
                if (eventType && eventData) {
                  handleSSEEvent(eventType, eventData, assistantDiv, streamCtx);
                }
                eventType = "";
                eventData = "";
              }
            }

            return processChunk();
          });
        }

        return processChunk();
      })
      .catch(function (err) {
        addMessage("error", err.message);
        finishStreaming(null, null);
      });
  }

  function handleSSEEvent(type, dataStr, assistantDiv, streamCtx) {
    var data;
    try {
      data = JSON.parse(dataStr);
    } catch (e) {
      return;
    }

    switch (type) {
      case "text":
        // Server sends: event: text, data: {"type":"text","text":"..."}
        var textContent = data.text || data.content || "";
        if (textContent) {
          streamCtx.rawText += textContent;
          // Show plain text while streaming for responsiveness
          assistantDiv.textContent = streamCtx.rawText;
          scrollToBottom();
        }
        break;
      case "thought":
        if (data.content) {
          addActivityItem("thought", data.content);
        }
        break;
      case "tool_call_start":
        // Server sends: event: tool_call_start, data: {"type":"tool_call_start","tool_call":{"id":"...","name":"..."}}
        var toolName = (data.tool_call && data.tool_call.name) || data.tool || "unknown";
        var toolInput = (data.tool_call && data.tool_call.input) ? JSON.stringify(data.tool_call.input) : "";
        addActivityItem("tool_call", toolName + (toolInput ? ": " + toolInput : ""));
        break;
      case "tool_call_delta":
        // Partial tool call arguments — log to activity
        if (data.text) {
          addActivityItem("tool_call", data.text.substring(0, 200));
        }
        break;
      case "tool_call_end":
        addActivityItem("tool_result", "Tool call completed");
        break;
      case "validation":
        addActivityItem("validation", (data.rule || "") + " [" + (data.status || "") + "] " + (data.message || ""));
        break;
      case "error":
        addActivityItem("error", data.message || data.error || "Unknown error");
        if (!streamCtx.rawText) {
          assistantDiv.textContent = "Error: " + (data.message || data.error || "Unknown error");
        }
        finishStreaming(assistantDiv, streamCtx);
        break;
      case "done":
        // Server sends: event: done, data: {"tokens":{...},"turns":N,"duration_ms":N}
        // or: event: done, data: {"type":"done","response":{"content":"...","tool_calls":[...],...}}
        var finalContent = "";
        if (data.response && data.response.content) {
          finalContent = data.response.content;
        } else if (data.message) {
          finalContent = data.message;
        } else if (data.output) {
          finalContent = data.output;
        }
        if (finalContent && !streamCtx.rawText) {
          streamCtx.rawText = finalContent;
        }
        // Show stats in activity
        if (data.tokens || data.turns || data.duration_ms) {
          var stats = [];
          if (data.turns) stats.push("turns: " + data.turns);
          if (data.duration_ms) stats.push("time: " + data.duration_ms + "ms");
          if (data.tokens) {
            var t = data.tokens;
            stats.push("tokens: " + (t.input || 0) + " in / " + (t.output || 0) + " out");
          }
          addActivityItem("done", stats.join(" | "));
        }
        finishStreaming(assistantDiv, streamCtx);
        break;
      default:
        // Unknown events are logged to activity
        addActivityItem(type, JSON.stringify(data).substring(0, 200));
    }
  }

  function finishStreaming(assistantDiv, streamCtx) {
    // Render accumulated markdown into the assistant message
    if (assistantDiv && streamCtx && streamCtx.rawText) {
      assistantDiv.innerHTML = renderMarkdown(streamCtx.rawText);
      scrollToBottom();
    }
    state.streaming = false;
    els.btnSend.disabled = false;
    els.input.focus();
  }

  // --- Event Listeners ---
  function initEvents() {
    els.btnSend.addEventListener("click", sendMessage);

    els.input.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

    // Auto-resize textarea
    els.input.addEventListener("input", function () {
      this.style.height = "auto";
      this.style.height = Math.min(this.scrollHeight, 200) + "px";
    });

    els.agentSelect.addEventListener("change", function () {
      state.currentAgent = this.value;
      state.sessionId = "";
      sessionStorage.removeItem("agentspec_session_id");
      els.messages.innerHTML = "";
      els.activityLog.innerHTML = "";
      if (state.currentAgent) {
        createSession(state.currentAgent);
        loadAgentSchema(state.currentAgent);
        addSystemMessage("Switched to agent: " + state.currentAgent);
      }
    });

    // API Key modal
    els.btnKey.addEventListener("click", function () {
      els.keyInput.value = state.apiKey;
      els.keyModal.classList.add("visible");
      els.keyInput.focus();
    });

    els.btnKeySave.addEventListener("click", function () {
      state.apiKey = els.keyInput.value.trim();
      sessionStorage.setItem("agentspec_api_key", state.apiKey);
      els.keyModal.classList.remove("visible");
      fetchAgents();
    });

    els.btnKeyClear.addEventListener("click", function () {
      state.apiKey = "";
      sessionStorage.removeItem("agentspec_api_key");
      els.keyInput.value = "";
      els.keyModal.classList.remove("visible");
      fetchAgents();
    });

    els.keyModal.addEventListener("click", function (e) {
      if (e.target === els.keyModal) {
        els.keyModal.classList.remove("visible");
      }
    });

    els.keyInput.addEventListener("keydown", function (e) {
      if (e.key === "Enter") {
        els.btnKeySave.click();
      }
    });

    // Clear activity log
    els.btnToggleActivity.addEventListener("click", function () {
      els.activityLog.innerHTML = "";
    });
  }

  // --- Init ---
  function init() {
    initRefs();
    initEvents();
    fetchAgents();
    addSystemMessage("Welcome to AgentSpec. Select an agent and start chatting.");
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
