//Messaging events and functions

//Import websocket conn from main.js
import {conn} from './main.js';

class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
};

function routeEvent(event) {
    if (event.type ==undefined) {
        alert("No type field in the event.");
        console.log("Event type is undefined");
        return;
    }
    switch (event.type) {
        case "new-message":
            console.log("New message: ", event.payload);
            break;
        case "error":
            console.log("Error: ", event.payload);
            break;
        default:
            alert("Unsupported event type: " + event.type);
            console.log("Unknown event type: ", event.type);
    }
}

function sendEvent(eventName, payload) {
    let event = new Event(eventName, payload);
    conn.send(JSON.stringify(event));
}

function sendMessage (message) {
    var newmessage = document.getElementById('new-message');
    if(newmessage != null) {
        sendEvent("new-message", newmessage.value);
        console.log("New Message Print: ", newmessage);
    }
    newmessage.value = "";
    return false
}


// Event listener for message sending
// This onsubmit uses the form to reload the page and the websocket gets reloaded which is why you cant see the message
// document.getElementById('new-message').onsubmit = sendMessage;
document.getElementById('message-submit').addEventListener('click', function(e) {
    e.preventDefault(); // prevent the default form behaviour, i.e. reloading the page
    sendMessage();
});
