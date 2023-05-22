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
    constructor(message, from, to){
        this.message = message;
        this.from = from;
        this.to = to;
    }
}

class ReceiveMessageEvent{
    constructor(message, from, to, sent){
        this.message = message;
        this.from = from;
        this.to = to;
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
        case "users-online":
            //Functionality to display online users
            const usersOnline = event;
            appendUsers(usersOnline);
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
    const formattedMsg = `<strong>${event.from} (${date.toLocaleTimeString()}): </strong>${event.message.replace(/\n/g, '<br>')}<br>`;
    let msgArea = document.getElementById('chat-messages');
    msgArea.innerHTML += formattedMsg;
     //Intrusive for the user attempting to read prevuous messages
    //because it scrolls to the bottom of the chat area.
    msgArea.scrollTop = msgArea.scrollHeight;
}

function sendEvent(eventName, payload) {
    let event = new Event(eventName, payload);
    conn.send(JSON.stringify(event));
}

export function sendMessage (message) {
    var newmessage = document.getElementById('new-message');
    //Get usernmae from local storage
    let username = localStorage.getItem('username');
    if(newmessage != null) {
        //Hard-coded value of the username needs to be changed???
        let outGoingMsg = new SendMessageEvent(newmessage.value, username);
        sendEvent("new-message", outGoingMsg);
        console.log("New Message Print: ", newmessage);
    }
    newmessage.value = "";
    return false
}

export function appendUsers(event) {
    console.log("Users: ", event.payload)
    let usersList = document.getElementById('users-list');
    usersList.innerHTML = "";
    // let users = JSON.parse(new TextDecoder().decode(new Uint8Array(event.payload))); // parse the JSON object
    let users = event.payload;

// Loop through the keys in the users object
for (let user in users) {
    // user will be the key (username), and users[user] will be the value (admin status)

    let newUser = document.createElement('li');
        
    // If the user is an admin, indicate that in the list
    newUser.textContent = user;
    
    const greenCircle = document.createElement("span");
    greenCircle.style.backgroundColor = "green";
    greenCircle.style.width = "10px";
    greenCircle.style.height = "10px";
    greenCircle.style.borderRadius = "50%";
    greenCircle.style.display = "inline-block";
    greenCircle.style.marginRight = "10px";
    
    newUser.appendChild(greenCircle);

    newUser.addEventListener("click", () => {
      openChatWindow(user);
      console.log("Clicked on user: " + user);
    });
    
    usersList.appendChild(newUser);
}
}


// Function to open a chat window between two users
function openChatWindow(user) {
  // Populate the chat window with the conversation history between the two users (if it exists)
  
}


// Function to get the conversation history between two users
// function getChatHistory(user, callback) {
//   // Send a message to the server to request the conversation history
//   const message = JSON.stringify({ type: "get_chat_history", user: user });
//   console.log("Sending message to backend!: " + message);
//   conn.send(message);

//   // Listen for the response from the server
//   conn.addEventListener("message", (event) => {
//     const message = JSON.parse(event.payload);
//     console.log("Message received from backend!: " + JSON.stringify(message));
//     if (message.type === "chat_history") {
//       const chatHistory = JSON.stringify(message);
//       const TempText = JSON.stringify(message);
//       console.log("TempText: " + TempText);
//       //for (const message of chatHistory) {
//       //  console.log("Message: " + JSON.stringify(message));
//       //}
//       callback(chatHistory);
//     }
//   });
// }
