import { sendMessage } from './chat.js';

export let conn;

window.onload = function () {
    // This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
    // document.getElementById('new-message').onsubmit = sendMessage;

    document.getElementById('message-submit').addEventListener('click', function(e) {
        e.preventDefault(); // prevent the default form submission
        sendMessage();
    });

    if(window["WebSocket"]) {
        console.log("WebSocket is supported by client browser!");
        conn = new WebSocket("wss://localhost:443/ws")  //Instead of 'localhost' you can use 'document.location.host' 
        console.log(conn);
    } else {
        alert("WebSocket NOT supported by client browser!");
        console.log("WebSocket NOT supported by client browser!");
    }
}