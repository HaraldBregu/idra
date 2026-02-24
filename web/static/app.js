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

    loadStatus();
    loadConfig();
    setInterval(loadStatus, 10000);
})();
