export let conn
import {routeEvent} from './chat.js'
import { Event } from './chat.js';

export function connectWebSocket(otp) {
    if(window["WebSocket"]) {
        console.log("WebSocket is supported by client browser!");
        // Request websocket connection with otp as query parameter
        conn = new WebSocket("wss://localhost:443/ws?otp="+ otp)  //Instead of 'localhost' you can use 'document.location.host' 
        console.log("Connection print: ",conn);

        conn.onopen = function(event) {
            console.log("Websocket connection established!");
        };

        conn.onclose = function(event) {
            console.log("Websocket connection closed!");
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