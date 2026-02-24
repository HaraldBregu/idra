(function () {
    "use strict";

    const $ = (sel) => document.querySelector(sel);

    let bearerToken = "";
    let tokenVisible = false;

    function api(method, path, body) {
        const opts = {
            method: method,
            headers: { "Content-Type": "application/json" },
        };
        if (bearerToken) {
            opts.headers["Authorization"] = "Bearer " + bearerToken;
        }
        if (body !== undefined) {
            opts.body = JSON.stringify(body);
        }
        return fetch(path, opts).then((r) => {
            if (!r.ok) throw new Error(r.statusText);
            return r.json();
        });
    }

    // --- Status ---

    function loadStatus() {
        api("GET", "/api/v1/status")
            .then((s) => {
                $("#version").textContent = s.version || "dev";
                $("#uptime").textContent = s.uptime || "—";
                $("#port").textContent = s.port || "—";
            })
            .catch(() => {});

        api("GET", "/api/v1/health")
            .then(() => {
                const el = $("#health");
                el.textContent = "ok";
                el.className = "value badge badge-ok";
            })
            .catch(() => {
                const el = $("#health");
                el.textContent = "error";
                el.className = "value badge badge-error";
            });
    }

    // --- Agents ---

    function loadAgents() {
        api("GET", "/api/v1/agents")
            .then((agents) => {
                const list = $("#agents-list");
                const select = $("#task-agent");

                if (!agents || agents.length === 0) {
                    list.innerHTML = '<p class="hint">No agents registered.</p>';
                    return;
                }

                list.innerHTML = "";
                // Preserve current selection
                const prev = select.value;
                select.innerHTML = '<option value="">Select an agent...</option>';

                agents.forEach((a) => {
                    // Agent card
                    const card = document.createElement("div");
                    card.className = "agent-card";
                    const stateClass =
                        a.state === "running"
                            ? "badge-ok"
                            : a.state === "failed"
                            ? "badge-error"
                            : "badge-unknown";
                    card.innerHTML =
                        '<div class="agent-header">' +
                        '  <span class="agent-name">' + esc(a.name) + "</span>" +
                        '  <span class="badge ' + stateClass + '">' + esc(a.state) + "</span>" +
                        "</div>" +
                        '<div class="agent-details">' +
                        '  <span class="agent-skills">Skills: ' + esc((a.skills || []).join(", ")) + "</span>" +
                        (a.port ? '  <span class="agent-port">Port: ' + a.port + "</span>" : "") +
                        (a.error ? '  <span class="agent-error">' + esc(a.error) + "</span>" : "") +
                        "</div>";
                    list.appendChild(card);

                    // Dropdown option
                    const opt = document.createElement("option");
                    opt.value = a.name;
                    opt.textContent = a.name + " (" + (a.skills || []).join(", ") + ")";
                    select.appendChild(opt);
                });

                // Restore selection
                if (prev) select.value = prev;
            })
            .catch(() => {
                $("#agents-list").innerHTML = '<p class="hint">Could not load agents.</p>';
            });
    }

    // --- Task execution ---

    $("#task-form").addEventListener("submit", function (e) {
        e.preventDefault();
        const agentName = $("#task-agent").value;
        const skill = $("#task-skill").value.trim();
        const input = $("#task-input").value.trim();

        if (!agentName) {
            showTaskStatus("Select an agent", true);
            return;
        }

        showTaskStatus("Executing...", false);
        $("#task-result").style.display = "none";

        api("POST", "/api/v1/agents/" + encodeURIComponent(agentName) + "/tasks", {
            skill: skill,
            input: input,
        })
            .then((res) => {
                showTaskStatus("Done", false);
                const output = $("#task-output");
                output.textContent = JSON.stringify(res, null, 2);
                $("#task-result").style.display = "block";
            })
            .catch((err) => {
                showTaskStatus("Error: " + err.message, true);
            });
    });

    function showTaskStatus(msg, isError) {
        const el = $("#task-status");
        el.textContent = msg;
        el.style.color = isError ? "#dc2626" : "";
        if (msg === "Done") {
            setTimeout(() => (el.textContent = ""), 3000);
        }
    }

    // --- Config ---

    function loadConfig() {
        api("GET", "/api/v1/config")
            .then((c) => {
                $("#cfg-port").value = c.port;
                $("#cfg-auto-open").checked = c.auto_open_browser;
                bearerToken = c.bearer_token || "";
            })
            .catch(() => {});
    }

    $("#config-form").addEventListener("submit", function (e) {
        e.preventDefault();
        const payload = {
            port: parseInt($("#cfg-port").value, 10),
            auto_open_browser: $("#cfg-auto-open").checked,
        };
        api("PUT", "/api/v1/config", payload)
            .then(() => {
                const el = $("#save-status");
                el.textContent = "Saved";
                setTimeout(() => (el.textContent = ""), 2000);
            })
            .catch((err) => {
                const el = $("#save-status");
                el.textContent = "Error: " + err.message;
                el.style.color = "#dc2626";
                setTimeout(() => {
                    el.textContent = "";
                    el.style.color = "";
                }, 3000);
            });
    });

    // --- Token ---

    $("#token-toggle").addEventListener("click", function () {
        const display = $("#token-display");
        if (tokenVisible) {
            display.textContent = "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022";
            this.textContent = "Show";
            tokenVisible = false;
        } else {
            display.textContent = bearerToken || "(none)";
            this.textContent = "Hide";
            tokenVisible = true;
        }
    });

    // --- Helpers ---

    function esc(str) {
        const d = document.createElement("div");
        d.textContent = str;
        return d.innerHTML;
    }

    // --- Init ---

    loadStatus();
    loadConfig();
    loadAgents();
    setInterval(loadStatus, 10000);
    setInterval(loadAgents, 10000);
})();
