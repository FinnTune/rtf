export let conn
import {routeEvent} from './chat.js'
import { Event } from './chat.js';
import { createMainHTML } from './mainHTML.js';

export function connectWebSocket(data) {
    if(window["WebSocket"]) {
        console.log("WebSocket is supported by client browser!");
        console.log("OTP from connectWS: ", data.otp);
        // Request websocket connection with otp as query parameter
        conn = new WebSocket("wss://localhost:443/ws?otp="+ data.otp)  //Instead of 'localhost' you can use 'document.location.host' 
        console.log("Connection print: ",conn);

        conn.onopen = function() {
            console.log("Websocket connection established!");
            // conn.send(JSON.stringify(data));
            //Create event to send to backend
            const eventObj = Object.assign(new Event("user-connect", data));
            console.log("Conn OnOpen data: ", data);
            conn.send(JSON.stringify(eventObj));
        };

        conn.onclose = function() {
            console.log("Websocket connection closed!");
            createMainHTML();
            let msg = document.getElementById("msg")
            msg.innerHTML = "You've been logged out."
            //Possible to reconnect here??? if accidentally closed because of network issues.
        };

        conn.onmessage = doOnMessage;
    } else {
        alert("WebSocket NOT supported by client browser!");
        console.log("WebSocket NOT supported by client browser!");
    };
}

function doOnMessage(event) {
    console.log("Event print: ", event);
    console.log("Event data print: ", event.data);
    const eventData = JSON.parse(event.data);
    const eventObj = Object.assign(new Event, eventData);
    routeEvent(eventObj);
};