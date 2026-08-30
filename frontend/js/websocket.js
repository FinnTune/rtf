export let conn
import {routeEvent} from './chat.js'
import { Event } from './chat.js';
import { checkLoginStatus } from './main.js';
import { showMessage } from './notify.js';

let expectedClose = false;
let reconnectAttempts = 0;
const maxReconnectAttempts = 3;

// Call before intentionally ending a session (e.g. logout) so the resulting
// close event isn't mistaken for a dropped connection and doesn't trigger a
// reconnect attempt.
export function disconnectWebSocket() {
    expectedClose = true;
    if (conn) {
        conn.close();
    }
}

export function connectWebSocket(data) {
    if(window["WebSocket"]) {
        expectedClose = false;
        // Request websocket connection with otp as query parameter
        conn = new WebSocket(`wss://${window.location.host}/ws?otp=${data.otp}`)

        conn.onopen = function() {
            reconnectAttempts = 0;
            //Create event to send to backend
            const eventObj = Object.assign(new Event("user-connect", data));
            conn.send(JSON.stringify(eventObj));
        };

        conn.onclose = function() {
            if (expectedClose) {
                expectedClose = false;
                return;
            }
            showMessage("Connection lost. Reconnecting...", "error");
            attemptReconnect();
        };

        conn.onmessage = doOnMessage;
    } else {
        showMessage("WebSocket is not supported by your browser.", "error");
    }
}

function attemptReconnect() {
    if (reconnectAttempts >= maxReconnectAttempts) {
        showMessage("Connection lost. Please refresh the page.", "error");
        return;
    }
    reconnectAttempts++;
    setTimeout(() => {
        checkLoginStatus().then((loggedIn) => {
            if (loggedIn) {
                showMessage("Reconnected.", "success");
            } else {
                showMessage("Your session has ended. Please log in again.", "error");
            }
        });
    }, 1000 * reconnectAttempts);
}

function doOnMessage(event) {
    const eventData = JSON.parse(event.data);
    const eventObj = Object.assign(new Event, eventData);
    routeEvent(eventObj);
}