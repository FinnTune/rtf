//Messaging events and functions

//Import websocket conn from main.js
import {conn} from './websocket.js';

export class Event {
    constructor(type, payload) {
        this.type = type;
        this.payload = payload;
    }
};

class SendMessageEvent {
    constructor(message, from){
        this.message = message;
        this.from = from;
    }
}

class ReceiveMessageEvent{
    constructor(message, from, sent){
        this.message = message;
        this.from = from;
        this.sent = sent;
    }
}

export function routeEvent(event) {
    if (event.type ==undefined) {
        alert("No type field in the event.");
        console.log("Event type is undefined");
        return;
    }
    switch (event.type) {
        case "sent-message":
            const messageEvent = Object.assign(new ReceiveMessageEvent, event.payload)
            console.log("New message: ", event.payload);
            appendChatMsg(messageEvent);
            break;
        case "error":
            console.log("Error: ", event.payload);
            break;
        default:
            alert("Unsupported event type: " + event.type);
            console.log("Unknown event type: ", event.type);
    }
}

function appendChatMsg(event) {
    var date = new Date(event.sent);
    const formattedMsg = `${date.toLocaleString()}: ${event.message}`;
    let msgArea = document.getElementById('chat-messages');
    msgArea.innerHTML = msgArea.innerHTML + "\n" + formattedMsg;
    //Intrusive for the user attempting to read prevuous messages
    //because it scrolls to the bottom of the chat area.
    // msgArea.scrollTop = msgArea.scrollHeight;
}

function sendEvent(eventName, payload) {
    let event = new Event(eventName, payload);
    conn.send(JSON.stringify(event));
}

export function sendMessage (message) {
    var newmessage = document.getElementById('new-message');
    if(newmessage != null) {
        //Hard-coded value of the username needs to be changed???
        let outGoingMsg = new SendMessageEvent(newmessage.value, "Client");
        sendEvent("new-message", outGoingMsg);
        console.log("New Message Print: ", newmessage);
    }
    newmessage.value = "";
    return false
}
