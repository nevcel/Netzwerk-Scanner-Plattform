"use strict";


// HTML-Elemente

const scanForm = document.getElementById("scan-form");
const scanTarget = document.getElementById("scan-target");
const scanButton = document.getElementById("scan-button");

const statusSection = document.getElementById("status-section");
const scanIdDisplay = document.getElementById("scan-id");
const scanTargetDisplay = document.getElementById("scan-target-display");
const scanStatusDisplay = document.getElementById("scan-status");
const scanMessage = document.getElementById("scan-message");
const scanProgressBar = document.getElementById("scan-progress-bar");

const resultsSection = document.getElementById("results-section");
const resultsContainer = document.getElementById("results-container");
const closeResultsButton = document.getElementById("close-results");

const historyContainer = document.getElementById("history-container");
const refreshHistoryButton = document.getElementById("refresh-history");



// Aktuell ausgewählter Scan

let activeScanId = null;


// Verhindert doppelte Statusabfragen für denselben Scan

const monitoredScans = new Set();


// Abstand zwischen den Statusabfragen

const POLLING_INTERVAL = 2000;


// REST-Kommunikation

async function apiRequest(path, options = {}) {

    const response = await fetch(
        path,
        options
    );

    const data =
        await response
            .json()
            .catch(() => null);

    if (!response.ok) {

        throw new Error(
            data?.error ||
            `HTTP-Fehler ${response.status}`
        );

    }

    return data;
}


// Scan starten

scanForm.addEventListener(
    "submit",
    async (event) => {

        event.preventDefault();

        const target =
            scanTarget
                .value
                .trim();

        if (!target) {

            showMessage(
                "Bitte eine IP-Adresse oder ein Netzwerk eingeben."
            );

            return;
        }


        scanButton.disabled = true;

        scanButton.textContent =
            "Scan wird gestartet ...";


        resultsSection.hidden = true;

        resultsContainer.replaceChildren();


        showMessage(
            "Scanauftrag wird übermittelt."
        );


        try {

            const job = await apiRequest(
                "/api/scans",
                {
                    method: "POST",

                    headers: {
                        "Content-Type":
                            "application/json"
                    },

                    body: JSON.stringify({
                        target: target
                    })
                }
            );


            activeScanId = job.id;


            displayScanStatus(job);


            void monitorScan(job.id);


        } catch (error) {

            setStatusVisual("failed");

            showMessage(
                `Fehler: ${error.message}`
            );


        } finally {

            scanButton.disabled = false;

            scanButton.textContent =
                "Scan starten";

        }
    }
);


// Scanstatus überwachen

async function monitorScan(scanId) {

    if (monitoredScans.has(scanId)) {
        return;
    }


    monitoredScans.add(scanId);


    try {

        while (true) {

            await new Promise(
                (resolve) => {

                    setTimeout(
                        resolve,
                        POLLING_INTERVAL
                    );

                }
            );


            const job =
                await apiRequest(
                    `/api/scans/${encodeURIComponent(scanId)}`
                );


            if (activeScanId === scanId) {

                displayScanStatus(job);

            }


            if (job.status === "completed") {

                if (activeScanId === scanId) {

                    showMessage(
                        "Scan erfolgreich abgeschlossen."
                    );

                    await loadScanResults(
                        scanId
                    );

                }


                await loadHistory();

                return;
            }


            if (job.status === "failed") {

                if (activeScanId === scanId) {

                    showMessage(
                        job.error ||
                        "Der Scan konnte nicht abgeschlossen werden."
                    );

                }


                await loadHistory();

                return;
            }

        }


    } catch (error) {

        if (activeScanId === scanId) {

            setStatusVisual("failed");

            showMessage(
                `Fehler: ${error.message}`
            );

        }


    } finally {

        monitoredScans.delete(scanId);

    }
}


// Scanstatus anzeigen

function displayScanStatus(job) {

    scanIdDisplay.textContent =
        job.id;

    scanTargetDisplay.textContent =
        job.target;


    setStatusVisual(
        job.status
    );


    const messages = {

        pending:
            "Scanauftrag wurde angenommen und wartet auf einen freien Worker.",

        running:
            "Netzwerk wird analysiert ...",

        completed:
            "Scan abgeschlossen.",

        failed:
            "Beim Scan ist ein Fehler aufgetreten."

    };


    showMessage(
        messages[job.status] ||
        job.status
    );

}


// Status optisch darstellen

function setStatusVisual(status) {

    const labels = {

        idle:
            "BEREIT",

        pending:
            "WARTEND",

        running:
            "LÄUFT",

        completed:
            "ABGESCHLOSSEN",

        failed:
            "FEHLER"

    };


    const progress = {

        idle:
            "0%",

        pending:
            "25%",

        running:
            "72%",

        completed:
            "100%",

        failed:
            "100%"

    };


    scanStatusDisplay.textContent =
        labels[status] ||
        status.toUpperCase();


    scanStatusDisplay.dataset.status =
        status;


    scanProgressBar.style.width =
        progress[status] ||
        "0%";

}


// Rückmeldung anzeigen

function showMessage(message) {

    statusSection.hidden = false;

    scanMessage.textContent =
        message;

}


// Scanergebnisse laden

async function loadScanResults(scanId) {

    const result =
        await apiRequest(
            `/api/scans/${encodeURIComponent(scanId)}/results`
        );


    if (activeScanId !== scanId) {
        return;
    }


    resultsContainer.replaceChildren();

    resultsSection.hidden = false;


    if (
        !result.hosts ||
        result.hosts.length === 0
    ) {

        const message =
            document.createElement("div");


        message.className =
            "empty-state";


        message.textContent =
            "Keine Hosts im Scanergebnis vorhanden.";


        resultsContainer.appendChild(
            message
        );

        return;
    }


    for (const host of result.hosts) {

        const hostArticle =
            document.createElement(
                "article"
            );


        hostArticle.className =
            "result-host";


        const heading =
            document.createElement(
                "h3"
            );


        heading.textContent =
            host.ip_address;


        const information =
            document.createElement(
                "p"
            );


        information.textContent =
            `Hostname: ${host.hostname || "–"} · Status: ${host.state}`;


        hostArticle.append(
            heading,
            information
        );


        if (
            !host.ports ||
            host.ports.length === 0
        ) {

            const message =
                document.createElement(
                    "p"
                );


            message.textContent =
                "Keine Ports erfasst.";


            hostArticle.appendChild(
                message
            );


        } else {

            const table =
                document.createElement(
                    "table"
                );


            const header =
                document.createElement(
                    "thead"
                );


            const headerRow =
                document.createElement(
                    "tr"
                );


            for (const label of [

                "Port",
                "Protokoll",
                "Status",
                "Dienst",
                "Produkt",
                "Version"

            ]) {

                const cell =
                    document.createElement(
                        "th"
                    );


                cell.textContent =
                    label;


                headerRow.appendChild(
                    cell
                );

            }


            header.appendChild(
                headerRow
            );


            const body =
                document.createElement(
                    "tbody"
                );


            for (const port of host.ports) {

                const row =
                    document.createElement(
                        "tr"
                    );


                const values = [

                    port.number,

                    port.protocol,

                    port.state,

                    port.service?.name ||
                        "–",

                    port.service?.product ||
                        "–",

                    port.service?.version ||
                        "–"

                ];


                for (const value of values) {

                    const cell =
                        document.createElement(
                            "td"
                        );


                    cell.textContent =
                        value;


                    row.appendChild(
                        cell
                    );

                }


                body.appendChild(
                    row
                );

            }


            table.append(
                header,
                body
            );


            hostArticle.appendChild(
                table
            );

        }


        resultsContainer.appendChild(
            hostArticle
        );

    }

}


// Scan-Historie laden

async function loadHistory() {

    try {

        const history =
            await apiRequest(
                "/api/scans"
            );


        historyContainer.replaceChildren();


        if (
            !history.scans ||
            history.scans.length === 0
        ) {

            const message =
                document.createElement(
                    "div"
                );


            message.className =
                "empty-state";


            message.textContent =
                "Noch keine Scanläufe vorhanden.";


            historyContainer.appendChild(
                message
            );

            return;
        }


        const list =
            document.createElement(
                "div"
            );


        list.className =
            "history-list";


        for (const scan of history.scans) {

            const row =
                document.createElement(
                    "div"
                );


            row.className =
                "history-row";


            const target =
                document.createElement(
                    "div"
                );


            target.className =
                "history-target";


            const targetValue =
                document.createElement(
                    "strong"
                );


            targetValue.textContent =
                scan.target;


            const targetMeta =
                document.createElement(
                    "span"
                );


            targetMeta.textContent =
                `Scan-ID: ${scan.id}`;


            target.append(
                targetValue,
                targetMeta
            );


            const date =
                document.createElement(
                    "div"
                );


            date.className =
                "history-date";


            date.textContent =
                new Date(
                    scan.created_at
                )
                    .toLocaleString(
                        "de-CH"
                    );


            const hosts =
                document.createElement(
                    "div"
                );


            hosts.className =
                "history-hosts";


            hosts.textContent =
                `${scan.host_count ?? 0} Host${
                    scan.host_count === 1
                        ? ""
                        : "s"
                }`;


            const status =
                document.createElement(
                    "div"
                );


            status.className =
                "history-state";


            status.dataset.status =
                scan.status;


            status.textContent =
                formatStatus(
                    scan.status
                );


            const button =
                document.createElement(
                    "button"
                );


            button.type =
                "button";


            button.className =
                "history-action";


            button.textContent =
                "Anzeigen";


            button.addEventListener(
                "click",
                async () => {

                    activeScanId =
                        scan.id;


                    try {

                        const job =
                            await apiRequest(
                                `/api/scans/${encodeURIComponent(scan.id)}`
                            );


                        displayScanStatus(
                            job
                        );


                        if (
                            job.status ===
                            "completed"
                        ) {

                            await loadScanResults(
                                scan.id
                            );


                        } else {

                            resultsSection.hidden =
                                true;


                            resultsContainer
                                .replaceChildren();


                            if (
                                job.status ===
                                    "pending" ||
                                job.status ===
                                    "running"
                            ) {

                                void monitorScan(
                                    scan.id
                                );

                            }

                        }


                    } catch (error) {

                        setStatusVisual(
                            "failed"
                        );


                        showMessage(
                            `Fehler: ${error.message}`
                        );

                    }

                }
            );


            row.append(
                target,
                date,
                hosts,
                status,
                button
            );


            list.appendChild(
                row
            );

        }


        historyContainer.appendChild(
            list
        );


    } catch (error) {

        const message =
            document.createElement(
                "div"
            );


        message.className =
            "error-state";


        message.textContent =
            `Historie konnte nicht geladen werden: ${error.message}`;


        historyContainer.replaceChildren(
            message
        );

    }

}


// Status für Historie formatieren

function formatStatus(status) {

    const labels = {

        pending:
            "WARTEND",

        running:
            "LÄUFT",

        completed:
            "ABGESCHLOSSEN",

        failed:
            "FEHLER"

    };


    return (
        labels[status] ||
        status.toUpperCase()
    );

}


// Historie manuell aktualisieren

refreshHistoryButton.addEventListener(
    "click",
    loadHistory
);


// Historie beim Laden der Seite anzeigen

loadHistory();

closeResultsButton.addEventListener(
    "click",
    () => {
        resultsSection.hidden = true;
    }
);