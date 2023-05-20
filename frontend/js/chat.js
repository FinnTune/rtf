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

function appendUsers(event) {
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

// Create a chat window element and append it to the DOM
const chatWindow = document.createElement("div");
chatWindow.className = "chat-window";
document.body.appendChild(chatWindow);

// Function to open a chat window between two users
function openChatWindow(user) {
  // Populate the chat window with the conversation history between the two users (if it exists)
  getChatHistory(user, (chatHistory) => {
    const history = JSON.parse(chatHistory);

    // Create a container for the chat messages
    const chatContainer = document.createElement("div");
    chatContainer.className = "chat-container";
    
    // Loop through each message in the chat history and display it in the chat window
    history.chathistory.forEach((message) => {
      const messageContainer = document.createElement("div");
      messageContainer.className = message.from === user.id ? "message sent" : "message received";
      
      const messageText = document.createElement("p");
      messageText.textContent = message.text;
      messageContainer.appendChild(messageText);
      
      chatContainer.appendChild(messageContainer);
    });
    
    // Create a container for the message input and send button
    const inputContainer = document.createElement("div");
    inputContainer.className = "input-container";
    
    // Create the message input field
    const messageInput = document.createElement("input");
    messageInput.type = "text";
    messageInput.placeholder = "Type your message...";
    inputContainer.appendChild(messageInput);
    
    // Create the send button
    const sendButton = document.createElement("button");
    sendButton.textContent = "Send";
    sendButton.addEventListener("click", () => {
      sendMessage(user, messageInput.value);
      console.log("Sending message: " + messageInput.value);
    });
    inputContainer.appendChild(sendButton);
    
    // Add the chat and input containers to the chat window
    chatWindow.innerHTML = "";
    chatWindow.appendChild(chatContainer);
    chatWindow.appendChild(inputContainer);
  });
}


// Function to get the conversation history between two users
function getChatHistory(user, callback) {
  // Send a message to the server to request the conversation history
  const message = JSON.stringify({ type: "get_chat_history", user: user });
  console.log("Sending message to backend!: " + message);
  socket.send(message);

  // Listen for the response from the server
  socket.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    console.log("Message received from backend!: " + JSON.stringify(message));
    if (message.type === "chat_history") {
      const chatHistory = JSON.stringify(message);
      const TempText = JSON.stringify(message);
      console.log("TempText: " + TempText);
      //for (const message of chatHistory) {
      //  console.log("Message: " + JSON.stringify(message));
      //}
      callback(chatHistory);
    }
  });
}
