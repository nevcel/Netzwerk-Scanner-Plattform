"use strict";


// Netzwerk-Scanner – Anwendungslogik
// Kommunikation mit dem Go-Backend über die Nginx-Weiterleitung.


// Zentrale Referenzen auf die benötigten HTML-Elemente.
const scanForm = document.getElementById("scan-form");
const scanTarget = document.getElementById("scan-target");
const scanButton = document.getElementById("scan-button");

const statusSection = document.getElementById("status-section");
const scanIdDisplay = document.getElementById("scan-id");
const scanTargetDisplay = document.getElementById("scan-target-display");
const scanStatusDisplay = document.getElementById("scan-status");
const scanMessage = document.getElementById("scan-message");

const resultsSection = document.getElementById("results-section");
const resultsContainer = document.getElementById("results-container");

const historyContainer = document.getElementById("history-container");
const refreshHistoryButton = document.getElementById("refresh-history");

// Die aktuell angezeigte Scan-ID bestimmt, welcher Auftrag
// im Status- und Ergebnisbereich dargestellt wird.
let activeScanId = null;

// Mehrere Scanaufträge können gleichzeitig überwacht werden.
// Das Set verhindert doppelte Statusabfragen für denselben Auftrag.
const monitoredScans = new Set();

const POLLING_INTERVAL = 2000;



// REST-Kommunikation


// apiRequest übernimmt die gemeinsamen HTTP-Anfragen.
// Bei einer fehlgeschlagenen Anfrage wird eine Fehlermeldung ausgelöst.
async function apiRequest(path, options = {}) {
    const response = await fetch(path, options);

    const data = await response.json().catch(() => null);

    if (!response.ok) {
        throw new Error(
            data?.error || `HTTP-Fehler ${response.status}`
        );
    }

    return data;
}



// Scan starten


scanForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    const target = scanTarget.value.trim();

    if (!target) {
        showMessage("Bitte eine IP-Adresse oder ein Netzwerk eingeben.");
        return;
    }

    // Während der Erstellung soll kein zweiter Auftrag
    // über denselben Button ausgelöst werden.
    scanButton.disabled = true;
    scanButton.textContent = "Scan wird gestartet ...";

    resultsSection.hidden = true;
    resultsContainer.replaceChildren();

    showMessage("Scanauftrag wird übermittelt.");

    try {
        const job = await apiRequest("/api/scans", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                target: target
            })
        });

        activeScanId = job.id;

        displayScanStatus(job);

        // Die Statusüberwachung läuft unabhängig vom Formular weiter.
        // Dadurch kann der Benutzer direkt einen weiteren Scan starten.
        void monitorScan(job.id);

    } catch (error) {
        showMessage(`Fehler: ${error.message}`);

    } finally {
        scanButton.disabled = false;
        scanButton.textContent = "Scan starten";
    }
});



// Asynchrone Statusüberwachung


// Jeder Scanauftrag wird unabhängig von den anderen überwacht.
// Nur der aktuell ausgewählte Scan aktualisiert die Statusanzeige.
async function monitorScan(scanId) {
    if (monitoredScans.has(scanId)) {
        return;
    }

    monitoredScans.add(scanId);

    try {
        while (true) {
            await new Promise((resolve) => {
                setTimeout(resolve, POLLING_INTERVAL);
            });

            const job = await apiRequest(
                `/api/scans/${encodeURIComponent(scanId)}`
            );

            // Ein Scan im Hintergrund darf die Anzeige eines
            // inzwischen ausgewählten anderen Scans nicht überschreiben.
            if (activeScanId === scanId) {
                displayScanStatus(job);
            }

            if (job.status === "completed") {
                if (activeScanId === scanId) {
                    showMessage("Scan erfolgreich abgeschlossen.");
                    await loadScanResults(scanId);
                }

                // Auch abgeschlossene Hintergrundscans
                // sollen in der Historie sichtbar werden.
                await loadHistory();

                return;
            }

            if (job.status === "failed") {
                if (activeScanId === scanId) {
                    showMessage(
                        job.error || "Der Scan konnte nicht abgeschlossen werden."
                    );
                }

                await loadHistory();

                return;
            }
        }

    } catch (error) {
        if (activeScanId === scanId) {
            showMessage(`Fehler: ${error.message}`);
        }

    } finally {
        monitoredScans.delete(scanId);
    }
}



// Statusanzeige


function displayScanStatus(job) {
    statusSection.hidden = false;

    scanIdDisplay.textContent = job.id;
    scanTargetDisplay.textContent = job.target;
    scanStatusDisplay.textContent = job.status;

    const messages = {
        pending: "Scanauftrag wurde angenommen.",
        running: "Netzwerk wird analysiert ...",
        completed: "Scan abgeschlossen.",
        failed: "Beim Scan ist ein Fehler aufgetreten."
    };

    showMessage(messages[job.status] || job.status);
}

function showMessage(message) {
    statusSection.hidden = false;
    scanMessage.textContent = message;
}



// Strukturierte Scanergebnisse darstellen


// Die Ergebnisse werden ausschliesslich über die REST-API
// geladen. Das Frontend verarbeitet kein Nmap-XML.
async function loadScanResults(scanId) {
    const result = await apiRequest(
        `/api/scans/${encodeURIComponent(scanId)}/results`
    );

    if (activeScanId !== scanId) {
        return;
    }

    resultsContainer.replaceChildren();
    resultsSection.hidden = false;

    if (!result.hosts || result.hosts.length === 0) {
        const message = document.createElement("p");
        message.textContent = "Keine Hosts im Scanergebnis vorhanden.";

        resultsContainer.appendChild(message);
        return;
    }

    // Jeder Host wird als eigener Ergebnisbereich aufgebaut.
    for (const host of result.hosts) {
        const hostArticle = document.createElement("article");
        hostArticle.className = "result-host";

        const heading = document.createElement("h3");
        heading.textContent = host.ip_address;

        const information = document.createElement("p");
        information.textContent =
            `Hostname: ${host.hostname || "–"} | Status: ${host.state}`;

        hostArticle.append(heading, information);

        if (!host.ports || host.ports.length === 0) {
            const message = document.createElement("p");
            message.textContent = "Keine Ports erfasst.";

            hostArticle.appendChild(message);

        } else {
            const table = document.createElement("table");

            const header = document.createElement("thead");
            const headerRow = document.createElement("tr");

            for (const label of [
                "Port",
                "Protokoll",
                "Status",
                "Dienst",
                "Produkt",
                "Version"
            ]) {
                const cell = document.createElement("th");
                cell.textContent = label;
                headerRow.appendChild(cell);
            }

            header.appendChild(headerRow);

            const body = document.createElement("tbody");

            for (const port of host.ports) {
                const row = document.createElement("tr");

                const values = [
                    port.number,
                    port.protocol,
                    port.state,
                    port.service?.name || "–",
                    port.service?.product || "–",
                    port.service?.version || "–"
                ];

                for (const value of values) {
                    const cell = document.createElement("td");

                    // textContent verhindert, dass erkannte Hostnamen
                    // oder Diensteinträge als HTML interpretiert werden.
                    cell.textContent = value;

                    row.appendChild(cell);
                }

                body.appendChild(row);
            }

            table.append(header, body);
            hostArticle.appendChild(table);
        }

        resultsContainer.appendChild(hostArticle);
    }
}



// Scan-Historie


// Die Historie wird aus PostgreSQL über das Backend geladen.
async function loadHistory() {
    try {
        const history = await apiRequest("/api/scans");

        historyContainer.replaceChildren();

        if (!history.scans || history.scans.length === 0) {
            const message = document.createElement("p");
            message.textContent = "Noch keine Scanläufe vorhanden.";

            historyContainer.appendChild(message);
            return;
        }

        const table = document.createElement("table");
        const header = document.createElement("thead");
        const headerRow = document.createElement("tr");

        for (const label of [
            "Datum",
            "Scanziel",
            "Status",
            "Hosts",
            "Aktion"
        ]) {
            const cell = document.createElement("th");
            cell.textContent = label;
            headerRow.appendChild(cell);
        }

        header.appendChild(headerRow);

        const body = document.createElement("tbody");

        for (const scan of history.scans) {
            const row = document.createElement("tr");

            const values = [
                new Date(scan.created_at).toLocaleString("de-CH"),
                scan.target,
                scan.status,
                scan.host_count
            ];

            for (const value of values) {
                const cell = document.createElement("td");
                cell.textContent = value;
                row.appendChild(cell);
            }

            // Über den Button kann ein früherer Scan
            // erneut aus der Datenbank geladen werden.
            const actionCell = document.createElement("td");
            const button = document.createElement("button");

            button.type = "button";
            button.textContent = "Anzeigen";

            button.addEventListener("click", async () => {
                activeScanId = scan.id;

                try {
                    const job = await apiRequest(
                        `/api/scans/${encodeURIComponent(scan.id)}`
                    );

                    displayScanStatus(job);

                    if (job.status === "completed") {
                        await loadScanResults(scan.id);

                    } else {
                        resultsSection.hidden = true;
                        resultsContainer.replaceChildren();

                        if (job.status === "pending" || job.status === "running") {
                            await monitorScan(scan.id);
                        }
                    }

                } catch (error) {
                    showMessage(`Fehler: ${error.message}`);
                }
            });

            actionCell.appendChild(button);
            row.appendChild(actionCell);

            body.appendChild(row);
        }

        table.append(header, body);
        historyContainer.appendChild(table);

    } catch (error) {
        historyContainer.textContent =
            `Historie konnte nicht geladen werden: ${error.message}`;
    }
}



// Initialisierung

// Beim Öffnen der Weboberfläche werden die bereits
// gespeicherten Scanläufe automatisch angezeigt.
refreshHistoryButton.addEventListener("click", loadHistory);

loadHistory();